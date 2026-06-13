import strictConfig from "./eslint.config.strict.js";
import { defineConfig } from "eslint/config";

export default defineConfig([
  ...strictConfig,
  {
    rules: {
      "@typescript-eslint/no-unused-vars": "off",
      "@typescript-eslint/no-explicit-any": "off",
      "react-hooks/exhaustive-deps": "off",
      "react-refresh/only-export-components": "off",
    },
  },
]);
