/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        forge: {
          black: "#000000",
          jet: "#050505",
          onyx: "#0b0d0f",
          void: "#050505",
          iron: "#0b0d0f",
          carbon: "#101214",
          coal: "#17191c",
          charcoal: "#17191c",
          slate: "#17191c",
          graphite: "#0b0d0f",
          steel: "#6f7782",
          platinum: "#e5e4e2",
          titanium: "#d1d5db",
          silver: "#c0c0c0",
          gray: "#c0c0c0",
          mist: "#c0c0c0",
          bone: "#e5e4e2",
          ash: "#e5e4e2",
          ultramarine: "#6f7782",
          electric: "#6f7782",
          sky: "#6f7782",
          gold: "#d1d5db",
          ember: "#d1d5db",
          emberSoft: "#d1d5db",
          violet: "#6f7782",
          royal: "#6f7782",
          bluevio: "#6f7782",
        },
      },
      fontFamily: {
        sans: ["Space Grotesk", "Sora", "Manrope", "Avenir Next", "Segoe UI", "ui-sans-serif", "sans-serif"],
        mono: ["JetBrains Mono", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
      boxShadow: {
        panel: "0 0 0 1px rgba(192, 192, 192, 0.12), 0 30px 80px rgba(0, 0, 0, 0.58)",
        header: "0 1px 0 0 rgba(229, 228, 226, 0.08)",
      },
      maxWidth: {
        work: "1440px",
      },
    },
  },
  plugins: [],
};
