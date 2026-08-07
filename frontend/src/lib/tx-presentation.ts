import { ArrowUpRight, Gift, RotateCcw, Settings2, Wallet } from "lucide-react"
import type { LucideIcon } from "lucide-react"
import type { TxChannel, TxType } from "@/lib/api/types"

/**
 * One place that decides how a movement looks. Every ledger row, receipt and
 * timeline entry reads from here, so the meaning of a colour stays constant
 * across the whole product.
 */
export interface TxPresentation {
  label: string
  icon: LucideIcon
  /** Text colour token. */
  tone: string
  /** Background chip token. */
  chip: string
  /** Sign applied to the amount from the *student's* point of view. */
  direction: 1 | -1 | 0
}

export const txPresentation: Record<TxType, TxPresentation> = {
  GRANT: {
    label: "Granted",
    icon: ArrowUpRight,
    tone: "text-grant",
    chip: "bg-grant-muted text-grant",
    direction: 1,
  },
  REDEMPTION: {
    label: "Redeemed",
    icon: Gift,
    tone: "text-redeem",
    chip: "bg-redeem-muted text-redeem-foreground",
    direction: -1,
  },
  GRANT_REVERSAL: {
    label: "Grant reversed",
    icon: RotateCcw,
    tone: "text-reverse",
    chip: "bg-reverse-muted text-reverse",
    direction: -1,
  },
  REDEMPTION_REVERSAL: {
    label: "Redemption reversed",
    icon: RotateCcw,
    tone: "text-reverse",
    chip: "bg-reverse-muted text-reverse",
    direction: 1,
  },
  VAULT_TOPUP: {
    label: "Vault top-up",
    icon: Wallet,
    tone: "text-primary",
    chip: "bg-accent text-accent-foreground",
    direction: 0,
  },
  ADJUSTMENT: {
    label: "Adjustment",
    icon: Settings2,
    tone: "text-muted-foreground",
    chip: "bg-muted text-muted-foreground",
    direction: 0,
  },
}

export const channelLabel: Record<TxChannel, string> = {
  QR: "QR scan",
  MANUAL: "Manual",
  BULK: "Bulk",
  SYSTEM: "System",
}

/**
 * Renders an amount the way the student experiences it: a grant reads "+3", a
 * redemption reads "−3", and a vault top-up (which touches no student) carries
 * no sign at all.
 */
export function signedAmount(type: TxType, amount: number): string {
  const { direction } = txPresentation[type]
  if (direction === 0) return String(amount)
  return `${direction > 0 ? "+" : "−"}${amount}`
}
