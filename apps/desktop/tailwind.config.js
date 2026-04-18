/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}", "../../packages/ui/src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        forge: {
          void: "#0b0c0f",
          iron: "#12141a",
          slate: "#1b1f27",
          steel: "#2a303c",
          mist: "#9aa3b2",
          ash: "#c7ccd6",
          ember: "#e24a1b",
          emberSoft: "#f06b3f",
        },
      },
      fontFamily: {
        sans: ["ui-sans-serif", "system-ui", "Segoe UI", "Inter", "Roboto", "Helvetica", "Arial", "sans-serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "Consolas", "Liberation Mono", "monospace"],
      },
      boxShadow: {
        panel: "0 0 0 1px rgba(255,255,255,0.05), 0 10px 40px rgba(0,0,0,0.45)",
        header: "0 1px 0 0 rgba(255,255,255,0.06)",
      },
      maxWidth: {
        work: "1440px",
      },
    },
  },
  plugins: [],
};
