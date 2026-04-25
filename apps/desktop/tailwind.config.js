/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        forge: {
          black: "#04080a",
          void: "#04080a",
          iron: "#0a1114",
          carbon: "#0f181c",
          charcoal: "#162126",
          slate: "#22323a",
          graphite: "#31444d",
          steel: "#31444d",
          gray: "#aebfc1",
          mist: "#d7e4e1",
          bone: "#edf6f2",
          ash: "#edf6f2",
          ultramarine: "#68e7ff",
          ember: "#68ffcc",
          sky: "#68e7ff",
          emberSoft: "#9dffd8",
          violet: "#ff9e4f",
          royal: "#9a5e1d",
          bluevio: "#ffd078",
        },
      },
      fontFamily: {
        sans: ["Space Grotesk", "Sora", "Manrope", "Avenir Next", "Segoe UI", "ui-sans-serif", "sans-serif"],
        mono: ["JetBrains Mono", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
      boxShadow: {
        panel: "0 0 0 1px rgba(165, 214, 219, 0.12), 0 30px 80px rgba(0, 0, 0, 0.58)",
        header: "0 1px 0 0 rgba(255,255,255,0.08)",
      },
      maxWidth: {
        work: "1440px",
      },
    },
  },
  plugins: [],
};
