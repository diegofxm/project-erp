import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// Config separada de vite.config.ts a propósito -- así el tooling de build/dev (tailwindcss,
// puerto del dev server, etc.) no se toca para nada al agregar tests, y viceversa. Solo el
// plugin de React hace falta acá (transformar JSX de los componentes bajo prueba).
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    // globals: false (default) a propósito -- así no hace falta tocar tsconfig.app.json para
    // que TS reconozca describe/it/expect como globales; cada test los importa explícito desde
    // "vitest", más simple que agregar "vitest/globals" a los types compartidos de todo src/.
    css: false,
  },
});
