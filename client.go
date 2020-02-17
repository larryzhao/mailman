package mailman

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
)

type AuthType int

const (
	AuthTypePlain AuthType = iota + 1
	// TODO: another auth type
)

type SMTPConfig struct {
	Server   string
	User     string
	Pass     string
	AuthType AuthType
}

type Client struct {
	config  *SMTPConfig
	smtpCli *smtp.Client
	ready   bool
}

func NewClient(config *SMTPConfig) (*Client, error) {
	client := &Client{
		config: config,
		ready:  false,
	}

	err := client.prepare()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (client *Client) Close() error {
	return client.smtpCli.Close()
}

func (client *Client) prepare() error {
	if client.ready {
		return nil
	}

	host, _, err := net.SplitHostPort(client.config.Server)
	if err != nil {
		return fmt.Errorf("split host port err on create smtp client: %w", err)
	}

	conn, err := tls.Dial("tcp", client.config.Server, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial err: %w", err)
	}

	smtpClient, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("err create stmp client over tls conn: %w", err)
	}

	err = smtpClient.Hello("aigauss.com")
	if err != nil {
		return fmt.Errorf("err on saying hello to smtp: %w", err)
	}

	var auth smtp.Auth
	switch client.config.AuthType {
	case AuthTypePlain:
		auth = smtp.PlainAuth("", client.config.User, client.config.Pass, host)
	}

	err = smtpClient.Auth(auth)
	if err != nil {
		return fmt.Errorf("err on authenticating with smtp: %w", err)
	}

	client.smtpCli = smtpClient
	client.ready = true
	return nil
}

func (client *Client) Deliver(msg *Message) error {
	var err error

	if !client.ready {
		err = client.prepare()
		if err != nil {
			return err
		}
	}

	err = client.do(msg)
	if err != nil {
		switch err.(type) {
		default:
			e := fmt.Errorf("calling smtp err %v: %w", err, err)
			log.Println(fmt.Sprintf("err happened %s, need reprepare", e.Error()))
			client.ready = false
			return e
		}
	}

	return nil
}

func (client *Client) do(msg *Message) error {
	var err error

	err = client.smtpCli.Mail(msg.From.Mail)
	if err != nil {
		return fmt.Errorf("err on smtp cmd Mail: %w", err)
	}

	for _, recipient := range msg.To {
		err = client.smtpCli.Rcpt(recipient.Mail)
		if err != nil {
			return fmt.Errorf("err on smtp cmd Rcpt with address %s: %w", recipient.Mail, err)
		}
	}

	wc, err := client.smtpCli.Data()
	if err != nil {
		return fmt.Errorf("err on smtp cmd Data: %w", err)
	}

	_, err = wc.Write(msg.SMTPBody())
	if err != nil {
		return fmt.Errorf("err on smtp write data: %w", err)
	}

	err = wc.Close()
	if err != nil {
		return fmt.Errorf("err on smtp close data wc: %w", err)
	}

	return nil
}
