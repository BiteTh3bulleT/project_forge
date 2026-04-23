/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        forge: {
          black: "#05070c",
          void: "#05070c",
          iron: "#0c1017",
          carbon: "#0c1017",
          charcoal: "#161b25",
          slate: "#161b25",
          graphite: "#242b39",
          steel: "#242b39",
          gray: "#b7bfd0",
          mist: "#b7bfd0",
          bone: "#eef1f8",
          ash: "#eef1f8",
          ultramarine: "#2a3cff",
          ember: "#2a3cff",
          sky: "#2ec8ff",
          emberSoft: "#2ec8ff",
          violet: "#8a4bff",
          royal: "#5b2eb8",
          bluevio: "#6f5bff",
        },
      },
      fontFamily: {
        sans: ["Space Grotesk", "Manrope", "Avenir Next", "Segoe UI", "ui-sans-serif", "sans-serif"],
        mono: ["JetBrains Mono", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
      boxShadow: {
        panel: "0 0 0 1px rgba(138, 75, 255, 0.12), 0 28px 80px rgba(0, 0, 0, 0.6)",
        header: "0 1px 0 0 rgba(255,255,255,0.08)",
      },
      maxWidth: {
        work: "1440px",
      },
    },
  },
  plugins: [],
};
