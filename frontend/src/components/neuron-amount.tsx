import { cn } from "@/lib/utils"

/**
 * The neuron glyph. Drawn rather than imported as an icon so it can inherit
 * currentColor and scale cleanly next to text at any size.
 */
export function NeuronIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      className={cn("size-4", className)}
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="3.2" fill="currentColor" />
      <circle cx="12" cy="12" r="8.2" stroke="currentColor" strokeWidth="1.4" opacity="0.45" />
      <circle cx="12" cy="3.8" r="1.5" fill="currentColor" />
      <circle cx="19.1" cy="16.1" r="1.5" fill="currentColor" />
      <circle cx="4.9" cy="16.1" r="1.5" fill="currentColor" />
    </svg>
  )
}

type Size = "sm" | "md" | "lg" | "xl" | "hero"

const sizeStyles: Record<Size, { text: string; icon: string; gap: string }> = {
  sm: { text: "text-sm", icon: "size-3.5", gap: "gap-1" },
  md: { text: "text-base", icon: "size-4", gap: "gap-1.5" },
  lg: { text: "text-2xl", icon: "size-5", gap: "gap-2" },
  xl: { text: "text-4xl", icon: "size-7", gap: "gap-2.5" },
  hero: { text: "text-6xl", icon: "size-10", gap: "gap-3" },
}

/**
 * The single most important number in the product. It is always rendered with
 * tabular figures so that columns of amounts in the ledger align, and always
 * paired with the glyph so a bare number is never mistaken for a grade.
 */
export function NeuronAmount({
  value,
  size = "md",
  signed = false,
  className,
  animate = false,
}: {
  value: number
  size?: Size
  /** Prefix with + / - and colour by direction, for ledger rows. */
  signed?: boolean
  className?: string
  animate?: boolean
}) {
  const s = sizeStyles[size]
  const sign = signed ? (value > 0 ? "+" : value < 0 ? "−" : "") : ""
  const tone = signed
    ? value > 0
      ? "text-grant"
      : value < 0
        ? "text-redeem"
        : "text-muted-foreground"
    : ""

  return (
    <span
      className={cn(
        "inline-flex items-center font-semibold tabular",
        s.text,
        s.gap,
        tone,
        animate && "animate-neuron-pop",
        className,
      )}
    >
      <NeuronIcon className={s.icon} />
      {sign}
      {Math.abs(value).toLocaleString()}
    </span>
  )
}
