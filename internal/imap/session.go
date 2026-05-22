package imap

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/google/uuid"
)

type session struct {
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	server   *Server
	logger   *slog.Logger
	user     *model.User
	mailbox  *model.Mailbox
	folder   *model.Folder
	state    sessionState
}

type sessionState int

const (
	stateNotAuthenticated sessionState = iota
	stateAuthenticated
	stateSelected
	stateLogout
)

func newSession(conn net.Conn, server *Server) *session {
	return &session{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		server: server,
		logger: server.logger,
		state:  stateNotAuthenticated,
	}
}

func (s *session) close() {
	s.conn.Close()
}

func (s *session) serve() {
	s.sendResponse("*", "OK PostOffice IMAP4rev1 ready")

	for s.state != stateLogout {
		s.conn.SetReadDeadline(time.Now().Add(30 * time.Minute))
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				s.logger.Debug("IMAP read error", "error", err)
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}

		s.handleCommand(line)
	}
}

func (s *session) handleCommand(line string) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		s.sendResponse("*", "BAD invalid command")
		return
	}

	tag := parts[0]
	cmd := strings.ToUpper(parts[1])
	var args string
	if len(parts) > 2 {
		args = parts[2]
	}

	switch cmd {
	case "CAPABILITY":
		s.cmdCapability(tag)
	case "NOOP":
		s.sendResponse(tag, "OK NOOP completed")
	case "LOGOUT":
		s.cmdLogout(tag)
	case "LOGIN":
		s.cmdLogin(tag, args)
	case "LIST":
		s.cmdList(tag, args)
	case "SELECT":
		s.cmdSelect(tag, args)
	case "EXAMINE":
		s.cmdSelect(tag, args) // same as SELECT but read-only
	case "FETCH":
		s.cmdFetch(tag, args)
	case "STORE":
		s.cmdStore(tag, args)
	case "SEARCH":
		s.cmdSearch(tag, args)
	case "CREATE":
		s.cmdCreate(tag, args)
	case "DELETE":
		s.cmdDelete(tag, args)
	case "EXPUNGE":
		s.cmdExpunge(tag)
	case "CLOSE":
		s.cmdClose(tag)
	case "IDLE":
		s.cmdIdle(tag)
	case "UID":
		s.cmdUID(tag, args)
	default:
		s.sendResponse(tag, "BAD unknown command")
	}
}

func (s *session) cmdCapability(tag string) {
	s.sendResponse("*", "CAPABILITY IMAP4rev1 IDLE LITERAL+ CHILDREN NAMESPACE")
	s.sendResponse(tag, "OK CAPABILITY completed")
}

func (s *session) cmdLogout(tag string) {
	s.sendResponse("*", "BYE PostOffice IMAP server closing connection")
	s.sendResponse(tag, "OK LOGOUT completed")
	s.state = stateLogout
}

func (s *session) cmdLogin(tag, args string) {
	if s.state != stateNotAuthenticated {
		s.sendResponse(tag, "BAD already authenticated")
		return
	}

	username, password := parseLoginArgs(args)
	if username == "" || password == "" {
		s.sendResponse(tag, "BAD invalid arguments")
		return
	}

	ctx := context.Background()
	user, err := s.server.authenticate(ctx, username, password)
	if err != nil {
		s.sendResponse(tag, "NO "+err.Error())
		return
	}

	s.user = user
	s.state = stateAuthenticated

	// Get first mailbox for this user
	mailboxes, _ := s.server.mailboxRepo.ListByUser(ctx, user.ID)
	if len(mailboxes) > 0 {
		s.mailbox = mailboxes[0]
	}

	s.logger.Info("IMAP login", "user", username)
	s.sendResponse(tag, "OK LOGIN completed")
}

func (s *session) cmdList(tag, args string) {
	if s.state < stateAuthenticated {
		s.sendResponse(tag, "NO not authenticated")
		return
	}

	ctx := context.Background()
	if s.mailbox == nil {
		s.sendResponse(tag, "OK LIST completed")
		return
	}

	folders, err := s.server.folderRepo.ListByMailbox(ctx, s.mailbox.ID)
	if err != nil {
		s.sendResponse(tag, "NO "+err.Error())
		return
	}

	for _, f := range folders {
		attrs := ""
		if f.SpecialUse != "" {
			attrs = f.SpecialUse + " "
		}
		if attrs != "" {
			s.sendResponse("*", fmt.Sprintf(`LIST (%s) "/" "%s"`, strings.TrimSpace(attrs), f.Name))
		} else {
			s.sendResponse("*", fmt.Sprintf(`LIST () "/" "%s"`, f.Name))
		}
	}
	s.sendResponse(tag, "OK LIST completed")
}

func (s *session) cmdSelect(tag, args string) {
	if s.state < stateAuthenticated {
		s.sendResponse(tag, "NO not authenticated")
		return
	}

	folderName := stripQuotes(args)
	ctx := context.Background()

	if s.mailbox == nil {
		s.sendResponse(tag, "NO no mailbox available")
		return
	}

	folder, err := s.server.folderRepo.GetByName(ctx, s.mailbox.ID, folderName)
	if err != nil {
		s.sendResponse(tag, "NO folder not found")
		return
	}

	s.folder = folder
	s.state = stateSelected

	s.sendResponse("*", fmt.Sprintf("%d EXISTS", folder.MessageCount))
	s.sendResponse("*", "0 RECENT")
	s.sendResponse("*", fmt.Sprintf("OK [UIDVALIDITY %d]", folder.UIDValidity))
	s.sendResponse("*", fmt.Sprintf("OK [UIDNEXT %d]", folder.UIDNext))
	s.sendResponse("*", fmt.Sprintf("OK [UNSEEN %d]", folder.UnseenCount))
	s.sendResponse("*", "FLAGS (\\Seen \\Answered \\Flagged \\Deleted \\Draft)")
	s.sendResponse(tag, fmt.Sprintf("OK [READ-WRITE] SELECT completed"))
}

func (s *session) cmdFetch(tag, args string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	seqSet, items := parseFetchArgs(args)
	ctx := context.Background()

	messages, _, err := s.server.messageRepo.ListByFolder(ctx, s.folder.ID, 0, 1000)
	if err != nil {
		s.sendResponse(tag, "NO "+err.Error())
		return
	}

	for i, msg := range messages {
		seqNum := i + 1
		if !inSeqSet(seqSet, seqNum) {
			continue
		}
		s.sendFetchResponse(seqNum, msg, items)
	}

	s.sendResponse(tag, "OK FETCH completed")
}

func (s *session) cmdUID(tag, args string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		s.sendResponse(tag, "BAD invalid UID command")
		return
	}

	subcmd := strings.ToUpper(parts[0])
	subargs := parts[1]

	switch subcmd {
	case "FETCH":
		s.cmdUIDFetch(tag, subargs)
	case "STORE":
		s.cmdStore(tag, subargs)
	case "SEARCH":
		s.cmdSearch(tag, subargs)
	default:
		s.sendResponse(tag, "BAD unknown UID subcommand")
	}
}

func (s *session) cmdUIDFetch(tag, args string) {
	seqSet, items := parseFetchArgs(args)
	ctx := context.Background()

	messages, _, err := s.server.messageRepo.ListByFolder(ctx, s.folder.ID, 0, 10000)
	if err != nil {
		s.sendResponse(tag, "NO "+err.Error())
		return
	}

	for i, msg := range messages {
		if !inSeqSet(seqSet, msg.UID) {
			continue
		}
		s.sendFetchResponse(i+1, msg, items)
	}

	s.sendResponse(tag, "OK UID FETCH completed")
}

func (s *session) sendFetchResponse(seqNum int, msg *model.Message, items string) {
	var parts []string

	itemsUpper := strings.ToUpper(items)

	if strings.Contains(itemsUpper, "UID") {
		parts = append(parts, fmt.Sprintf("UID %d", msg.UID))
	}
	if strings.Contains(itemsUpper, "FLAGS") {
		flags := buildFlags(msg)
		parts = append(parts, fmt.Sprintf("FLAGS (%s)", flags))
	}
	if strings.Contains(itemsUpper, "ENVELOPE") {
		env := buildEnvelope(msg)
		parts = append(parts, fmt.Sprintf("ENVELOPE %s", env))
	}
	if strings.Contains(itemsUpper, "RFC822.SIZE") || strings.Contains(itemsUpper, "BODY") {
		parts = append(parts, fmt.Sprintf("RFC822.SIZE %d", msg.SizeBytes))
	}
	if strings.Contains(itemsUpper, "BODY[]") || strings.Contains(itemsUpper, "RFC822") {
		ctx := context.Background()
		raw, err := s.server.msgStore.Get(ctx, msg.StorageKey)
		if err == nil {
			parts = append(parts, fmt.Sprintf("BODY[] {%d}\r\n%s", len(raw), string(raw)))
		}
	}
	if strings.Contains(itemsUpper, "INTERNALDATE") {
		parts = append(parts, fmt.Sprintf(`INTERNALDATE "%s"`, msg.Date.Format("02-Jan-2006 15:04:05 -0700")))
	}

	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("UID %d", msg.UID))
		flags := buildFlags(msg)
		parts = append(parts, fmt.Sprintf("FLAGS (%s)", flags))
	}

	s.sendResponse("*", fmt.Sprintf("%d FETCH (%s)", seqNum, strings.Join(parts, " ")))
}

func (s *session) cmdStore(tag, args string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	// Parse: sequence +FLAGS (\Seen)
	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 3 {
		s.sendResponse(tag, "BAD invalid STORE arguments")
		return
	}

	flagStr := extractFlags(parts[2])
	ctx := context.Background()

	messages, _, _ := s.server.messageRepo.ListByFolder(ctx, s.folder.ID, 0, 10000)
	seqSet := parts[0]

	for i, msg := range messages {
		if !inSeqSet(seqSet, i+1) {
			continue
		}
		flags := parseFlags(flagStr)
		s.server.messageRepo.UpdateFlags(ctx, msg.ID, flags)
	}

	s.sendResponse(tag, "OK STORE completed")
}

func (s *session) cmdSearch(tag, args string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	ctx := context.Background()
	messages, _, _ := s.server.messageRepo.ListByFolder(ctx, s.folder.ID, 0, 10000)

	var uids []string
	argsUpper := strings.ToUpper(args)

	for _, msg := range messages {
		match := true
		if strings.Contains(argsUpper, "UNSEEN") && msg.IsSeen {
			match = false
		}
		if strings.Contains(argsUpper, "SEEN") && !strings.Contains(argsUpper, "UNSEEN") && !msg.IsSeen {
			match = false
		}
		if strings.Contains(argsUpper, "FLAGGED") && !msg.IsFlagged {
			match = false
		}
		if match {
			uids = append(uids, strconv.Itoa(msg.UID))
		}
	}

	s.sendResponse("*", "SEARCH "+strings.Join(uids, " "))
	s.sendResponse(tag, "OK SEARCH completed")
}

func (s *session) cmdCreate(tag, args string) {
	if s.state < stateAuthenticated {
		s.sendResponse(tag, "NO not authenticated")
		return
	}

	folderName := stripQuotes(args)
	ctx := context.Background()

	folder := &model.Folder{
		ID:          uuid.New(),
		MailboxID:   s.mailbox.ID,
		Name:        folderName,
		UIDValidity: 1,
		UIDNext:     1,
	}

	if err := s.server.folderRepo.Create(ctx, folder); err != nil {
		s.sendResponse(tag, "NO "+err.Error())
		return
	}

	s.sendResponse(tag, "OK CREATE completed")
}

func (s *session) cmdDelete(tag, args string) {
	if s.state < stateAuthenticated {
		s.sendResponse(tag, "NO not authenticated")
		return
	}

	folderName := stripQuotes(args)
	ctx := context.Background()

	folder, err := s.server.folderRepo.GetByName(ctx, s.mailbox.ID, folderName)
	if err != nil {
		s.sendResponse(tag, "NO folder not found")
		return
	}

	if err := s.server.folderRepo.Delete(ctx, folder.ID); err != nil {
		s.sendResponse(tag, "NO "+err.Error())
		return
	}

	s.sendResponse(tag, "OK DELETE completed")
}

func (s *session) cmdExpunge(tag string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	ctx := context.Background()
	messages, _, _ := s.server.messageRepo.ListByFolder(ctx, s.folder.ID, 0, 10000)

	expunged := 0
	for i, msg := range messages {
		if msg.IsDeleted {
			s.server.messageRepo.Delete(ctx, msg.ID)
			s.sendResponse("*", fmt.Sprintf("%d EXPUNGE", i+1-expunged))
			expunged++
		}
	}

	s.sendResponse(tag, "OK EXPUNGE completed")
}

func (s *session) cmdClose(tag string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	// Expunge silently
	ctx := context.Background()
	messages, _, _ := s.server.messageRepo.ListByFolder(ctx, s.folder.ID, 0, 10000)
	for _, msg := range messages {
		if msg.IsDeleted {
			s.server.messageRepo.Delete(ctx, msg.ID)
		}
	}

	s.folder = nil
	s.state = stateAuthenticated
	s.sendResponse(tag, "OK CLOSE completed")
}

func (s *session) cmdIdle(tag string) {
	if s.state != stateSelected {
		s.sendResponse(tag, "NO no mailbox selected")
		return
	}

	s.sendContinuation("idling")

	// Wait for DONE or timeout
	s.conn.SetReadDeadline(time.Now().Add(29 * time.Minute))
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return
	}

	if strings.TrimSpace(strings.ToUpper(line)) == "DONE" {
		s.sendResponse(tag, "OK IDLE terminated")
	}
}

func (s *session) sendResponse(tag, msg string) {
	fmt.Fprintf(s.writer, "%s %s\r\n", tag, msg)
	s.writer.Flush()
}

func (s *session) sendContinuation(msg string) {
	fmt.Fprintf(s.writer, "+ %s\r\n", msg)
	s.writer.Flush()
}

// Helper functions

func parseLoginArgs(args string) (string, string) {
	// Handle quoted strings: LOGIN "user" "pass"
	parts := splitQuoted(args)
	if len(parts) >= 2 {
		return stripQuotes(parts[0]), stripQuotes(parts[1])
	}
	return "", ""
}

func splitQuoted(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func parseFetchArgs(args string) (string, string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return args, ""
}

func inSeqSet(seqSet string, num int) bool {
	if seqSet == "*" || seqSet == "1:*" {
		return true
	}

	for _, part := range strings.Split(seqSet, ",") {
		if strings.Contains(part, ":") {
			rangeParts := strings.SplitN(part, ":", 2)
			start, _ := strconv.Atoi(rangeParts[0])
			var end int
			if rangeParts[1] == "*" {
				end = 999999
			} else {
				end, _ = strconv.Atoi(rangeParts[1])
			}
			if num >= start && num <= end {
				return true
			}
		} else {
			if part == "*" {
				return true
			}
			n, _ := strconv.Atoi(part)
			if n == num {
				return true
			}
		}
	}
	return false
}

func buildFlags(msg *model.Message) string {
	var flags []string
	if msg.IsSeen {
		flags = append(flags, "\\Seen")
	}
	if msg.IsAnswered {
		flags = append(flags, "\\Answered")
	}
	if msg.IsFlagged {
		flags = append(flags, "\\Flagged")
	}
	if msg.IsDeleted {
		flags = append(flags, "\\Deleted")
	}
	if msg.IsDraft {
		flags = append(flags, "\\Draft")
	}
	return strings.Join(flags, " ")
}

func buildEnvelope(msg *model.Message) string {
	date := msg.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700")
	subject := msg.Subject
	from := msg.FromAddress
	to := strings.Join(msg.ToAddresses, ", ")
	msgID := msg.MessageID
	inReplyTo := msg.InReplyTo

	return fmt.Sprintf(`("%s" "%s" (("%s")) (("%s")) NIL (("%s")) "%s" "%s")`,
		date, subject, from, from, to, msgID, inReplyTo)
}

func extractFlags(s string) string {
	start := strings.Index(s, "(")
	end := strings.Index(s, ")")
	if start >= 0 && end > start {
		return s[start+1 : end]
	}
	return s
}

func parseFlags(flagStr string) repository.MessageFlags {
	flags := repository.MessageFlags{}
	upper := strings.ToUpper(flagStr)

	if strings.Contains(upper, "\\SEEN") {
		t := true
		flags.IsSeen = &t
	}
	if strings.Contains(upper, "\\ANSWERED") {
		t := true
		flags.IsAnswered = &t
	}
	if strings.Contains(upper, "\\FLAGGED") {
		t := true
		flags.IsFlagged = &t
	}
	if strings.Contains(upper, "\\DELETED") {
		t := true
		flags.IsDeleted = &t
	}
	if strings.Contains(upper, "\\DRAFT") {
		t := true
		flags.IsDraft = &t
	}
	return flags
}
