/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        forge: {
          void: "#05070d",
          iron: "#0d1118",
          slate: "#151b26",
          steel: "#1f2734",
          mist: "#9da6b6",
          ash: "#cfd5e0",
          carbon: "#1f2838",
          ember: "#4963ff",
          emberSoft: "#7e91ff",
        },
      },
      fontFamily: {
        sans: ["Space Grotesk", "Manrope", "Avenir Next", "Segoe UI", "ui-sans-serif", "sans-serif"],
        mono: ["JetBrains Mono", "SFMono-Regular", "Menlo", "Consolas", "monospace"],
      },
      boxShadow: {
        panel: "0 0 0 1px rgba(120, 141, 255, 0.08), 0 28px 80px rgba(0, 0, 0, 0.55)",
        header: "0 1px 0 0 rgba(255,255,255,0.06)",
      },
      maxWidth: {
        work: "1440px",
      },
    },
  },
  plugins: [],
};
