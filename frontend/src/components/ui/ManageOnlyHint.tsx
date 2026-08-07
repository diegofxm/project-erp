// Nota que acompaña cualquier sección/botón deshabilitado por useCanManage — texto centralizado
// acá para que no se desincronice la redacción entre las distintas secciones que lo usan.
export function ManageOnlyHint() {
  return (
    <p className="text-xs text-(--text-muted)">Solo un administrador o dueño de la empresa puede editar esto.</p>
  );
}
