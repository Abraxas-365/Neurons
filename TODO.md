# Neurons - TODO

Items sourced from Manuel's feedback (Aug 1, 2026) and deployment needs.

## Manuel's Feedback

- [ ] **Leaderboard should show student's course**
  In the teacher Dashboard (`DashboardPage.tsx`), the leaderboard should display
  which course/classroom each student belongs to next to their name.

- [ ] **Student info visible on QR screen**
  When the student shows their QR code (`MyWalletPage.tsx:MyQRDialog`), the
  screen should also prominently display the student's name, their team, and
  their classroom/course. So the professor can visually confirm identity before
  scanning.

- [ ] **Lockable benefits with variable pricing**
  On the student's benefits view, some benefits should appear locked (with a
  padlock icon) until the professor activates them. The exchange rates for
  benefits can change during class or across the semester, so:
  - Benefits need an `active/locked` toggle the teacher controls.
  - Locked benefits show to students but are greyed out / show a lock icon
    and hide the price.
  - Teachers need an easy way to edit benefit values (neuron cost) on the fly.

- [ ] **Share code fallback (student -> teacher)**
  On the student QR dialog, add a "Share" button that lets the student send
  the text code to the teacher via SMS/WhatsApp/etc. using the Web Share API
  (`navigator.share`). Fallback to clipboard copy if Share API is unavailable.
  The teacher already has manual text entry on `ScanPage.tsx`.

