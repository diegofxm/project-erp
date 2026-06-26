// Lee un archivo (ej. el .p12 del certificado) y devuelve su contenido en base64 puro, sin el
// prefijo "data:<mime>;base64," que agrega FileReader.readAsDataURL — apidian espera el campo
// certificate_base64 sin ese prefijo (decodifica con encoding/base64 directo en Go).
export function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      const base64 = result.slice(result.indexOf(",") + 1);
      resolve(base64);
    };
    reader.onerror = () => reject(reader.error ?? new Error("No se pudo leer el archivo"));
    reader.readAsDataURL(file);
  });
}
