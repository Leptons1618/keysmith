# Design Review - SSH Key Manager

Date: 2026-08-21 · Mode: `review` · Surface: PySide6 desktop app, `main.py` (post-redesign "workbench ledger")

## Scores

| Lens | Score | Key finding |
|---|---|---|
| First impression | 8/10 | Distinct paper-and-ink voice; welcome pane leaves a large dead zone on tall windows |
| Hierarchy | 7/10 | Section scaffolding reads instantly; long key names break the identity block they anchor |
| Color voice | 8/10 | Ink-only strong fill, green reserved for verified states; disabled state nearly invisible |
| Type voice | 7/10 | Authored small-caps labels, mono confined to key material; workbench scale is flat, meta wrapping is ragged |
| Interaction feel | 6/10 | Workflow-order tab path, named busy state with cancel; splitter can erase the whole workbench |
| **Total** | **36/50** | |

## Findings

| # | Severity | Discipline | Location | Before | After | Why |
|---|---|---|---|---|---|---|
| 1 | HIGH | Surface | `main.py` `_build_key_details_page` (`key_details_name`), `_refresh_keys` (list items) | Long key names clip mid-glyph with no ellipsis; header name overflows its box by 372 px (measured with `id_work_github_production_deploy_2026_very_long_name`) and crowds the state badges | Elide the display name with `QFontMetrics.elidedText` (middle elide), show the full name in a tooltip on both the header label and list items, and give the name row stretch priority over the badges | Identity is the anchor of this screen; with realistic production key names the operator loses the exact object being operated on, in both the inventory and the inspector |
| 2 | MEDIUM | Interaction | `main.py` `_build_ui` vertical splitter (`v_split`, sizes `[760, 130]`) | Dragging the activity-strip handle collapses the workbench to 0 height (observed sizes `[0, 618]`) with no way to recover except blind re-drag | Call `setChildrenCollapsible(False)` on `v_split` and set a minimum height (~380 px) on the workbench side so the strip can shrink but never swallow the pane | One careless drag erases the entire working surface; collapse-by-default is a trap, not a feature |
| 3 | MEDIUM | Type | `main.py` `_build_key_details_page` `meta_grid` (fingerprint, private key rows) | Wrapped hash/path values hang back under the label column; the fingerprint row renders an orphaned fragment ("256") on its own line before the hash starts | Render each meta value as one self-contained wrapped line (bold inline prefix + mono value in a single word-wrapped QLabel) so continuation lines align under the value, not the label | Reading the fingerprint is the trust ritual of this product; a broken record layout undermines the verification moment it exists for |
| 4 | LOW | Writing | `main.py` dialog titles ("Key Generation", "Add key to agent", "Delete key", "SSH Agent") | Dialog titles use Title Case while the redesigned UI is sentence case | Sentence case everywhere: "Key generation failed", "Add key to agent" | Voice consistency; the modals are the last Title Case holdouts |
| 5 | LOW | Color | `STYLESHEET` `QPushButton:disabled` / `QLineEdit:disabled` (`#a5a196` on `#f3f1ec`, 2.29:1) | During any worker run, disabled controls fade to near-invisibility on the paper background | Darken disabled text toward `#767268` (~4.0:1) or add a dashed border cue so disabled remains perceivable | Every operation disables half the UI; WCAG exempts disabled text, but "barely there" reads as broken in an instrument this quiet |

## Smell lens

Clean. No category reflexes detected: no blue-violet CTAs, no gradient heroes, no card grids, no emoji, no terminal-cliche dark theme. The small-caps letter-spaced group titles are a deliberate system applied consistently, not a borrowed tell.

## Considered but rejected

| Location | Candidate | Rejected because |
|---|---|---|
| Keys rail | Enforce a wider minimum rail width | Splitter is already non-collapsible horizontally and the 915 px window minimum holds (measured); the rail degrades gracefully at min width |
| State badges | Add icons next to badge text for non-color redundancy | Badges already carry text labels ("IN AGENT", "TESTED OK"); the text does the work, icons would add noise to a text-first instrument |
| Delete flow | Replace the confirm dialog with undo-based trash | Deletion is irreversible file removal from `~/.ssh`; per undo-beats-confirm rules, irreversible deletes are exactly where confirm belongs |
| Passphrase mismatch | Inline field error instead of modal | Rare-path validation inside an app whose error grammar is already modal; consistent, low harm |
| Window floor | Flag the 915x703 minimum as a responsive failure | Desktop Qt instrument, fits a 1366x768 laptop with margin; a 320 px web-reflow rule does not transfer here |

## Verification

Ran (offscreen PySide6 render, sandboxed `HOME` with realistic seeded keys/state):

- Rendered and inspected: populated details, welcome, empty, busy overlay, agent ON/OFF badge states, focused line-edit and buttons
- Pixel-sampled: busy scrim `#484743` uniformly over paper; badge-on tint `#e7f0e9` applies
- Tab walk: list -> generation form -> actions -> test -> header, cycling cleanly
- Focus visibility: pixel diffs of 14k-43k changed px between focused/unfocused controls
- WCAG math on live stylesheet pairs: body 15-17:1, muted 4.8-5.1:1, accents/steps/badges 5.1-7.1:1, disabled 2.29:1
- Geometry: honors 1080x720, min 915x703; both splitters live (horizontal `[343, 706]`)
- Long-name probe: 372 px header overflow, clipped list row (screenshot reviewed)
- Collapse probe: workbench reduced to `[0, 618]` via `moveSplitter`
- Compile check plus end-to-end generate smoke test with real `ssh-keygen`

Not verified (stated gaps, not findings):

- Hover rendering: synthetic hover events produced zero pixel diff offscreen; QSS hover styles are defined but visually unconfirmed
- Screen-reader announcements: Qt accessibility names/object info untested
- Platform variance: Linux offscreen only; Windows/macOS native styling unchecked

## Verdict

**Block** - finding 1 (HIGH) is standing. Fix the identity elision first; findings 2-3 are the next pass. Recommended modes in order: `/design surface`, `/design interaction`, `/design typeset`, `/design finish`.
