Compilar apidian/
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o apidian-linux-amd64 ./cmd/server

Compilar frontend/
npm run build