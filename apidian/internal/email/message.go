package email

// Attachment es un archivo adjunto a un Message — el remitente ya conoce su tipo (PDF, XML),
// no se adivina por extensión, mismo criterio que issuers.LogoContentType.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Message es un correo a enviar.
type Message struct {
	To          string
	CC          []string // destinatarios en copia; nil o vacío = sin CC
	Subject     string
	BodyText    string
	BodyHTML    string
	Attachments []Attachment
}
