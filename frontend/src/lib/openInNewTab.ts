// Abre una URL en una pestaña nueva usando un <a> programático en lugar de window.open.
// window.open con blob: URLs puede ser bloqueado por el bloqueador de popups del navegador,
// lo que hace que el blob se abra en la pestaña actual y descargue la app de React.
export function openInNewTab(url: string): void {
  const a = document.createElement("a");
  a.href = url;
  a.target = "_blank";
  a.rel = "noopener noreferrer";
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  // Revocar el blob URL después de un minuto para liberar memoria.
  if (url.startsWith("blob:")) {
    setTimeout(() => URL.revokeObjectURL(url), 60_000);
  }
}
