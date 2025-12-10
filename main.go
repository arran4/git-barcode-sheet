package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// Scanner always appends a newline (<CR> / Enter).
// All Code values are complete git commands and DO NOT include newline characters.

type GitCmd struct {
	Code        string // exact text encoded in the barcode (no newline)
	Label       string // short label under barcode
	Description string // explanation under the label
}

// 40 git CLI commands -> 4 x 10 grid, all self-contained (no editing needed).
var Commands = []GitCmd{
	// --- Status / inspection ---
	{"git status", "git status", "Show working tree status."},
	{"git status -sb", "git status -sb", "Short, branch-aware status."},
	{"git diff", "git diff", "Diff unstaged changes."},
	{"git diff --staged", "git diff --staged", "Diff staged changes."},

	// --- Staging / restoring ---
	{"git add .", "git add .", "Stage all changes in current repo."},
	{"git add -p", "git add -p", "Interactive patch staging."},
	{"git restore .", "git restore .", "Discard unstaged changes in files."},
	{"git restore --staged .", "git restore --staged .", "Unstage all changes."},

	// --- Common commit messages ---
	{"git commit -m \"Initial commit\"", "Initial commit", "Create an initial commit."},
	{"git commit -m \"Update README\"", "Update README", "Commit README changes."},
	{"git commit -m \"Fix bug\"", "Fix bug", "Commit a bugfix."},
	{"git commit -m \"Refactor code\"", "Refactor code", "Commit refactor changes."},

	// --- Generic commit / log helpers ---
	{"git commit -m \"WIP\"", "WIP commit", "Quick work-in-progress commit."},
	{"git log --oneline --graph --decorate --all", "Pretty log", "Compact decorated log graph."},
	{"git log --oneline", "Log oneline", "Short one-line commit history."},
	{"git show", "git show", "Show details of the latest commit."},

	// --- Stash ---
	{"git stash", "git stash", "Stash uncommitted changes."},
	{"git stash pop", "stash pop", "Apply and drop latest stash."},
	{"git stash list", "stash list", "List all stashes."},
	{"git stash drop", "stash drop", "Drop latest stash."},

	// --- Branching & navigation ---
	{"git branch", "git branch", "List local branches."},
	{"git branch -vv", "git branch -vv", "Branches with tracking info."},
	{"git checkout -", "git checkout -", "Switch to previous branch."},
	{"git reflog", "git reflog", "Show reference log for HEAD history."},

	// --- Sync / remotes ---
	{"git fetch --all --prune", "fetch --all", "Fetch all remotes and prune."},
	{"git pull", "git pull", "Pull from current upstream."},
	{"git push", "git push", "Push current HEAD to upstream."},
	{"git push --set-upstream origin HEAD", "push -u origin HEAD", "Push and set upstream."},

	// --- Tags / metadata ---
	{"git tag", "git tag", "List tags."},
	{"git tag -l", "git tag -l", "List tags (pattern-capable)."},
	{"git remote -v", "git remote -v", "List remotes and URLs."},
	{"git config --list", "git config --list", "Show all Git config entries."},

	// --- Search / history helpers ---
	{"git grep -n \"TODO\"", "grep TODO", "Search TODO in tracked files."},
	{"git shortlog -sn", "shortlog -sn", "Author summary (commits per author)."},
	{"git rev-parse --show-toplevel", "repo root", "Show path to repo root."},
	{"git rev-parse --abbrev-ref HEAD", "current branch", "Show current branch name."},

	// --- Cleanup / caution ---
	{"git status --ignored", "status ignored", "Status including ignored files."},
	{"git diff --stat", "diff --stat", "Diff summary (per-file stats)."},
	{"git clean -fd", "clean -fd", "Danger: remove untracked files & dirs."},
	{"git submodule update --init --recursive", "submodules", "Init and update submodules."},
}

// font cache so we only parse Go Regular once per size.
var fontCache = map[float64]font.Face{}

func main() {
	// A4 @ 300 DPI
	const dpi = 300
	const a4WidthInches = 8.27
	const a4HeightInches = 11.69

	width := int(a4WidthInches * dpi)
	height := int(a4HeightInches * dpi)

	dc := gg.NewContext(width, height)

	// Background
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Tighter margins to reduce white space
	margin := 60.0

	// Title (larger font)
	dc.SetColor(color.Black)
	dc.SetFontFace(mustGoRegularFace(28))
	title := "Git Barcode Sheet – One Scan = One Command"
	dc.DrawStringAnchored(title, float64(width)/2, margin/2, 0.5, 0.5)

	// Layout: 4 columns, N rows
	cols := 4
	rows := int(math.Ceil(float64(len(Commands)) / float64(cols)))

	top := margin
	bottom := float64(height) - margin
	left := margin
	right := float64(width) - margin

	cellWidth := (right - left) / float64(cols)
	cellHeight := (bottom - top) / float64(rows)

	// In each cell:
	// - If command is short: draw wide Code128 barcode
	// - If command is long: draw square-ish QR

	// Threshold (characters) for "short" vs "long" commands
	const shortCmdMaxLen = 26

	for i, cmd := range Commands {
		col := i % cols
		row := i / cols

		x := left + float64(col)*cellWidth
		y := top + float64(row)*cellHeight

		cx := x + cellWidth/2

		// Light cell boundary
		dc.SetLineWidth(0.6)
		dc.SetColor(color.RGBA{R: 220, G: 220, B: 220, A: 255})
		dc.DrawRectangle(x, y, cellWidth, cellHeight)
		dc.Stroke()

		// Choose barcode type based on length
		if len(cmd.Code) <= shortCmdMaxLen {
			// --- Code128 for short commands ---
			barWidth := int(cellWidth * 0.9)
			barHeight := int(cellHeight * 0.45)

			raw, err := code128.Encode(cmd.Code)
			if err != nil {
				log.Printf("Code128 encode error for %q: %v", cmd.Code, err)
				continue
			}

			scaled, err := barcode.Scale(raw, barWidth, barHeight)
			if err != nil {
				log.Printf("Code128 scale error for %q: %v", cmd.Code, err)
				continue
			}

			bx := cx - float64(scaled.Bounds().Dx())/2
			by := y + 6
			dc.DrawImage(scaled, int(bx), int(by))

			// Label & description
			labelY := by + float64(barHeight) + 10

			dc.SetColor(color.Black)
			dc.SetFontFace(mustGoRegularFace(13))
			label := cmd.Label
			if label == "" {
				label = cmd.Code
			}
			dc.DrawStringAnchored(label, cx, labelY, 0.5, 0)

			descY := labelY + 14
			dc.SetFontFace(mustGoRegularFace(10))
			dc.DrawStringWrapped(cmd.Description, x+8, descY, 0, 0, cellWidth-16, 1.3, gg.AlignCenter)

		} else {
			// --- QR for long commands ---
			qrSize := int(math.Min(cellWidth*0.75, cellHeight*0.6))

			raw, err := qr.Encode(cmd.Code, qr.M, qr.Auto)
			if err != nil {
				log.Printf("QR encode error for %q: %v", cmd.Code, err)
				continue
			}

			scaled, err := barcode.Scale(raw, qrSize, qrSize)
			if err != nil {
				log.Printf("QR scale error for %q: %v", cmd.Code, err)
				continue
			}

			bx := cx - float64(scaled.Bounds().Dx())/2
			by := y + 8
			dc.DrawImage(scaled, int(bx), int(by))

			labelY := by + float64(qrSize) + 10

			dc.SetColor(color.Black)
			dc.SetFontFace(mustGoRegularFace(13))
			label := cmd.Label
			if label == "" {
				label = cmd.Code
			}
			dc.DrawStringAnchored(label, cx, labelY, 0.5, 0)

			descY := labelY + 14
			dc.SetFontFace(mustGoRegularFace(10))
			dc.DrawStringWrapped(cmd.Description, x+8, descY, 0, 0, cellWidth-16, 1.3, gg.AlignCenter)
		}
	}

	// --- Footer: repo QR + text --- (kept inside the page)
	footerText := "https://github.com/arran4/git-barcode-sheet"

	footerRaw, err := qr.Encode(footerText, qr.M, qr.Auto)
	if err != nil {
		log.Printf("QR encode error for footer: %v", err)
	} else {
		// Keep the QR comfortably inside the bottom margin
		footerSize := int(math.Min(float64(width)*0.16, margin*0.9))

		footerScaled, err := barcode.Scale(footerRaw, footerSize, footerSize)
		if err != nil {
			log.Printf("QR scale error for footer: %v", err)
		} else {
			// Place QR above bottom margin, centered horizontally
			fbX := float64(width)/2 - float64(footerScaled.Bounds().Dx())/2
			fbY := float64(height) - margin - float64(footerSize) + 4
			dc.DrawImage(footerScaled, int(fbX), int(fbY))

			// Footer text just above page bottom
			textY := float64(height) - 10
			dc.SetColor(color.Black)
			dc.SetFontFace(mustGoRegularFace(11))
			dc.DrawStringAnchored(footerText, float64(width)/2, textY, 0.5, 0)
		}
	}

	out := "git-barcode-sheet-a4.png"
	if err := dc.SavePNG(out); err != nil {
		log.Fatalf("failed to save PNG: %v", err)
	}

	fmt.Println("Saved:", out)
}

// mustGoRegularFace returns a Go Regular font.Face at the given size,
// always using the embedded goregular TTF.
func mustGoRegularFace(size float64) font.Face {
	if face, ok := fontCache[size]; ok {
		return face
	}

	fnt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatalf("failed to parse goregular TTF: %v", err)
	}

	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatalf("failed to create goregular face (size=%.1f): %v", size, err)
	}

	fontCache[size] = face
	return face
}
