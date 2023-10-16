import path from "path";
import { fileURLToPath } from "url";

const __dirname = fileURLToPath(import.meta.url);

export default {
  entry: "./src/index.js",
  output: {
    filename: "index.js",
    path: path.resolve(__dirname, "../out"),
  },
};
