import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// cleanup() desmonta el árbol de React entre tests -- sin esto, dos tests que rendericen la
// misma página dejarían nodos duplicados en el DOM de jsdom y los queries (getByRole, etc.)
// empezarían a fallar con "found multiple elements" de forma intermitente según el orden.
afterEach(() => {
  cleanup();
  localStorage.clear();
});
