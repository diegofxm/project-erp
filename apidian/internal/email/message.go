package email

// Attachment es un archivo adjunto a un Message — el remitente ya conoce su tipo (PDF, XML),
// no se adivina por extensión, mismo criterio que issuers.LogoContentType.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Message es un correo a enviar. Un solo destinatario por ahora — el caso real que se viene es
// un cliente por factura; si alguna vez hace falta CC/BCC/varios destinatarios, se extiende
// entonces.
type Message struct {
	To          string
	Subject     string
	BodyText    string
	BodyHTML    string
	Attachments []Attachment
}
