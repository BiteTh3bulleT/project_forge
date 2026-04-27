/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        forge: {
          black: "#030516",
          void: "#030516",
          iron: "#090d2b",
          carbon: "#0c1030",
          charcoal: "#121848",
          slate: "#18205f",
          graphite: "#253184",
          steel: "#3240a6",
          gray: "#b5bee8",
          mist: "#dfe5ff",
          bone: "#f3f5ff",
          ash: "#f3f5ff",
          ultramarine: "#120a8f",
          electric: "#3157ff",
          accent: "#3157ff",
          ember: "#ff6d8a",
          sky: "#9aacff",
          emberSoft: "#ff9ab0",
          violet: "#6678ff",
          royal: "#120a8f",
          bluevio: "#c4ccff",
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
