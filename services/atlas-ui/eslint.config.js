import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";
import eslintConfigPrettier from "eslint-config-prettier/flat";
import { defineConfig, globalIgnores } from "eslint/config";

export default defineConfig([
  globalIgnores(["dist", "node_modules", ".next", "app", "components", "context", "hooks", "lib", "services", "types", "tests"]),
  {
    files: ["src/**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
      eslintConfigPrettier,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_" },
      ],
    },
  },
  {
    files: [
      "src/components/providers/**/*.{ts,tsx}",
      "src/components/ui/**/*.{ts,tsx}",
      "src/context/**/*.{ts,tsx}",
      // Column-definition modules colocate a row-action component alongside
      // the exported column config (accounts-columns.tsx, characters-columns.tsx).
      "src/pages/**/*-columns.tsx",
      // Error-boundary modules colocate a fallback component, a context,
      // and/or a wrapping HOC alongside the boundary component itself.
      "src/components/**/*ErrorBoundary.tsx",
      // React context modules colocate the Provider component and its
      // consumer hook, same pattern as src/context/**.
      "src/components/**/*Context.tsx",
    ],
    rules: {
      "react-refresh/only-export-components": "off",
    },
  },
]);
