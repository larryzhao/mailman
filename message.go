package mailman

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

type Address struct {
	Name string
	Mail string
}

type Message struct {
	From        *Address
	To          []*Address
	Subject     string
	ContentText string
	Attachments []Attachment
}

func (msg *Message) SMTPBody() []byte {
	buf := &bytes.Buffer{}
	msg.writeHeaders(buf)
	msg.writeMixed(buf)
	return buf.Bytes()
}

func (msg *Message) writeHeaders(buf *bytes.Buffer) {
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")))
	buf.WriteString(fmt.Sprintf("From: \"%s\" <%s>\r\n", msg.From.Name, msg.From.Mail))

	recipientsText := make([]string, len(msg.To))
	for idx, addr := range msg.To {
		recipientsText[idx] = fmt.Sprintf("\"%s\" <%s>", addr.Name, addr.Mail)
	}
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(recipientsText, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buf.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
}

func (msg *Message) writeMixed(buf *bytes.Buffer) {
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"MixedBoundaryString\"\r\n\r\n"))
	buf.WriteString("--MixedBoundaryString\r\n")
	msg.writeRelated(buf)
	for _, attachment := range msg.Attachments {
		msg.writeAttachment(buf, attachment)
	}
	buf.WriteString("--MixedBoundaryString--")
}

func (msg *Message) writeRelated(buf *bytes.Buffer) {
	buf.WriteString("Content-Type: multipart/related; boundary=\"RelatedBoundaryString\"\r\n\r\n")
	buf.WriteString("--RelatedBoundaryString\r\n")
	msg.writeAlternative(buf)
	buf.WriteString("--RelatedBoundaryString--\r\n\r\n")
}

func (msg *Message) writeAlternative(buf *bytes.Buffer) {
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"AlternativeBoundaryString\"\r\n\r\n")
	buf.WriteString("--AlternativeBoundaryString\r\n")
	buf.WriteString("Content-Type: text/plain;charset=\"utf-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(msg.ContentText)
	buf.WriteString("\r\n\r\n")
	// TODO: add HTML
	buf.WriteString("--AlternativeBoundaryString--\r\n\r\n")
}

func (msg *Message) writeAttachment(buf *bytes.Buffer, attachment Attachment) {
	reader := attachment.Data()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		// TODO: log err and do nothing
		return
	}

	buf.WriteString("--MixedBoundaryString\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: %s;name=\"%s\"\r\n", attachment.ContentType(), attachment.Filename()))
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: attachment;filename=\"%s\"\r\n", attachment.Filename()))

	encodedData := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encodedData, data)
	buf.Write(encodedData)
	buf.WriteString("\r\n")
}
