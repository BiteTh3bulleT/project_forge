/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    "../../packages/ui/src/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        forge: {
          black: "#030303",
          void: "#050505",
          iron: "#090a0b",
          carbon: "#101214",
          charcoal: "#171a1d",
          slate: "#20252a",
          graphite: "#2a3036",
          steel: "#3c444b",
          gray: "#b8bec4",
          mist: "#dfe3e6",
          bone: "#f4f5f6",
          ash: "#f8f8f8",
          ultramarine: "#101214",
          electric: "#ff7a18",
          accent: "#ff7a18",
          ember: "#f97316",
          sky: "#cbd1d6",
          emberSoft: "#ffb84d",
          violet: "#8f969d",
          royal: "#15171a",
          titanium: "#e2e5e7",
        },
      },
      fontFamily: {
        sans: [
          "Space Grotesk",
          "Sora",
          "Manrope",
          "Avenir Next",
          "Segoe UI",
          "ui-sans-serif",
          "sans-serif",
        ],
        mono: [
          "JetBrains Mono",
          "SFMono-Regular",
          "Menlo",
          "Consolas",
          "monospace",
        ],
      },
      boxShadow: {
        panel:
          "0 0 0 1px rgba(220, 224, 226, 0.1), 0 30px 80px rgba(0, 0, 0, 0.6)",
        header: "0 1px 0 0 rgba(255,255,255,0.08)",
      },
      maxWidth: {
        work: "1440px",
      },
    },
  },
  plugins: [],
};
