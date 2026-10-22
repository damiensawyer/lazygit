<div align="center">
<sup>Special thanks to:</sup>
<br>
<br>
<a href="https://www.warp.dev/?utm_source=github&utm_medium=referral&utm_campaign=lazygit_20231023">
  <div>
    <img src="https://github.com/warpdotdev/brand-assets/blob/main/Github/Sponsor/Warp-Github-LG-02.png?raw=true" width="400" alt="Warp">
  </div>
  <b>Warp, the intelligent terminal</b>
  <br>
  <b>Available for macOS and Linux</b>
  <br>
  <div>
    <sup>Visit warp.dev to learn more.</sup>
  </div>
</a>
<br>
<hr>
<a href="https://tuple.app/lazygit">
  <div>
    <img src="../assets/tuple.png" width="400" alt="Tuple">
  </div>
  <b>Tuple, the premier screen sharing app for developers on macOS and Windows.</b>
</a>
<br>
<hr>
<br>
<a href="https://www.subble.com">
  <div>
    <img src="../assets/subble.webp" width="400" alt="Subble">
  </div>
  <b>I (Jesse) co-founded Subble to save your company time and money by finding unused and over-provisioned SaaS licences. Check it out!</b>
</a>
<br>

<hr>
</div>

<p align="center">
  <img width="536" src="https://user-images.githubusercontent.com/8456633/174470852-339b5011-5800-4bb9-a628-ff230aa8cd4e.png">
</p>

<div align="center">

A simple terminal UI for git commands — with GitHub Actions, a repo-files browser, and more
<br/>

</div>

## Why this fork

The upstream [lazygit](https://github.com/jesseduffield/lazygit) repository stopped accepting pull requests. This fork preserves the upstream codebase and adds features that the upstream maintainers declined to merge. It stays in sync with upstream's `master` branch — you can merge upstream changes in whenever you want.

> **Note:** This repo is currently closed to PRs, but I may open it up in the future. If you have a feature or fix you'd like to see, open an issue and I'll consider it.

## What's different from upstream

### GitHub Actions integration

Lazygit can show your repository's GitHub Actions workflow runs and do most of what `gh run` and `gh workflow` do — all from inside the TUI.

**To use it:** open a repo that's pushed to GitHub and press `<Ctrl+A>`. The runs list replaces the commits panel; press `Esc` to get back.

| Key | Action |
|-----|--------|
| `enter` | View a run's jobs |
| `o` | Open in browser |
| `r` | Re-run (all jobs or just failed ones) |
| `c` | Cancel or force-cancel |
| `d` | Delete the run |
| `f` | Filter runs by branch, workflow, status, event, or commit |
| `w` | Workflows: run, enable/disable, edit, or filter |
| `a` | Download the run's artifacts |
| `y` | Copy URL, ID, commit hash, or branch |
| `e` | Open a job's log in your editor |

The token is taken from your `gh` CLI login — if you're not logged in, lazygit offers a device-login flow (same as `gh auth login --web`) or you can paste a personal access token. The token is stored in `gh`'s `hosts.yml`, so `gh` uses it too.

See [docs-master/GitHub_Actions.md](docs-master/GitHub_Actions.md) for the full reference.

### Browse all files at any commit

Press `E` on any commit, branch, tag, stash, reflog entry, or remote branch to open a file-tree panel that shows the **entire repository** as it existed at that point — not just the files the commit changed.

| Key | Action |
|-----|--------|
| `enter` | Toggle a directory or focus a file's content in the main view |
| `` ` `` | Toggle between tree and flat layout |
| `-` / `=` | Collapse / expand all directories |
| `/` | Filter the tree by path |
| `c` | **Restore to worktree**: replace files in your working tree with their state at this commit |
| `S` | **Save**: write a copy of the file(s) as they existed at this commit |
| `y` | Copy menu: name, path, content, blob hash |
| `o` / `e` | Open/edit the *current working tree* version |

See [docs-master/Browsing_All_Files_At_Commit.md](docs-master/Browsing_All_Files_At_Commit.md) for the full reference.

### Syntax-highlighted file contents

File contents shown in the repo-files panel (and other file views) are syntax-highlighted, making it much easier to read code at a glance.

### Full file diff view toggle

When viewing a diff, press `Alt+}` to toggle between the unified diff and the full file content. This is useful when you want to see the complete file rather than just the changed lines.

### Searchable keybindings menu

The keybindings help menu is now searchable by typing. Just start typing and it filters to matching keybindings, making it easy to find what you're looking for without scanning the whole list.

### Side panel width adjustment

Adjust the width of side panels using the same shortcuts in both normal and half-screen modes. The width is persisted per-repository, so each repo can have its own preferred panel size.

### Final version toggle

Toggle between showing a diff and showing the final version of changed files. This is useful when you want to review the complete file content rather than just the changes.

### Copy keybinding customization

Confirmation popups can now choose what their copy keybinding copies — for example, a "Yes" button on a delete confirmation can copy the item's name instead of its URL.

## Table of contents

- [Why this fork](#why-this-fork)
- [What's different from upstream](#whats-different-from-upstream)
  - [GitHub Actions integration](#github-actions-integration)
  - [Browse all files at any commit](#browse-all-files-at-any-commit)
  - [Syntax-highlighted file contents](#syntax-highlighted-file-contents)
  - [Full file diff view toggle](#full-file-diff-view-toggle)
  - [Searchable keybindings menu](#searchable-keybindings-menu)
  - [Side panel width adjustment](#side-panel-width-adjustment)
  - [Final version toggle](#final-version-toggle)
  - [Copy keybinding customization](#copy-keybinding-customization)
- [Table of contents](#table-of-contents)
- [Sponsors](#sponsors)
- [Elevator Pitch](#elevator-pitch)
- [Features](#features)
- [Tutorials](#tutorials)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [Donate](#donate)
- [FAQ](#faq)
- [Shameless Plug](#shameless-plug)
- [Alternatives](#alternatives)

Lazygit is not my fulltime job but it is a hefty part time job so if you want to support the project please consider [sponsoring me](https://github.com/sponsors/jesseduffield)

## Sponsors

<p align="center">
 Maintenance of this project is made possible by all the <a href="https://github.com/jesseduffield/lazygit/graphs/contributors">contributors</a> and <a href="https://github.com/sponsors/jesseduffield">sponsors</a>. If you'd like to sponsor this project and have your avatar or company logo appear below <a href="https://github.com/sponsors/jesseduffield">click here</a>. 💙
</p>

<p align="center">
<!-- sponsors --><a href="https://github.com/intabulas"><img src="https:&#x2F;&#x2F;github.com&#x2F;intabulas.png" width="60px" alt="User avatar: Mark Lussier" /></a><a href="https://github.com/peppy"><img src="https:&#x2F;&#x2F;github.com&#x2F;peppy.png" width="60px" alt="User avatar: Dean Herbert" /></a><a href="https://github.com/piot"><img src="https:&#x2F;&#x2F;github.com&#x2F;piot.png" width="60px" alt="User avatar: Peter Bjorklund" /></a><a href="https://github.com/oliverguenther"><img src="https:&#x2F;&#x2F;github.com&#x2F;oliverguenther.png" width="60px" alt="User avatar: Oliver Günther" /></a><a href="https://github.com/pawanjay176"><img src="https:&#x2F;&#x2F;github.com&#x2F;pawanjay176.png" width="60px" alt="User avatar: Pawan Dhananjay" /></a><a href="https://github.com/naoey"><img src="https:&#x2F;&#x2F;github.com&#x2F;naoey.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/carstengehling"><img src="https:&#x2F;&#x2F;github.com&#x2F;carstengehling.png" width="60px" alt="User avatar: Carsten Gehling" /></a><a href="https://github.com/Xetera"><img src="https:&#x2F;&#x2F;github.com&#x2F;Xetera.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/nartc"><img src="https:&#x2F;&#x2F;github.com&#x2F;nartc.png" width="60px" alt="User avatar: Chau Tran" /></a><a href="https://github.com/matejcik"><img src="https:&#x2F;&#x2F;github.com&#x2F;matejcik.png" width="60px" alt="User avatar: matejcik" /></a><a href="https://github.com/lucatume"><img src="https:&#x2F;&#x2F;github.com&#x2F;lucatume.png" width="60px" alt="User avatar: theAverageDev (Luca Tumedei)" /></a><a href="https://github.com/IvanZuy"><img src="https:&#x2F;&#x2F;github.com&#x2F;IvanZuy.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/ava1ar"><img src="https:&#x2F;&#x2F;github.com&#x2F;ava1ar.png" width="60px" alt="User avatar: Aliaksandr Stelmachonak" /></a><a href="https://github.com/pedropombeiro"><img src="https:&#x2F;&#x2F;github.com&#x2F;pedropombeiro.png" width="60px" alt="User avatar: Pedro Pombeiro" /></a><a href="https://github.com/bburgy"><img src="https:&#x2F;&#x2F;github.com&#x2F;bburgy.png" width="60px" alt="User avatar: Burgy Benjamin" /></a><a href="https://github.com/JoeKlemmer"><img src="https:&#x2F;&#x2F;github.com&#x2F;JoeKlemmer.png" width="60px" alt="User avatar: Joe Klemmer" /></a><a href="https://github.com/tobi"><img src="https:&#x2F;&#x2F;github.com&#x2F;tobi.png" width="60px" alt="User avatar: Tobias Lütke" /></a><a href="https://github.com/benbfortis"><img src="https:&#x2F;&#x2F;github.com&#x2F;benbfortis.png" width="60px" alt="User avatar: Ben Beaumont" /></a><a href="https://github.com/jakewarren"><img src="https:&#x2F;&#x2F;github.com&#x2F;jakewarren.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/tgpholly"><img src="https:&#x2F;&#x2F;github.com&#x2F;tgpholly.png" width="60px" alt="User avatar: Holly" /></a><a href="https://github.com/Tommylans"><img src="https:&#x2F;&#x2F;github.com&#x2F;Tommylans.png" width="60px" alt="User avatar: Tom Lanser" /></a><a href="https://github.com/socketbox"><img src="https:&#x2F;&#x2F;github.com&#x2F;socketbox.png" width="60px" alt="User avatar: Casey Boettcher" /></a><a href="https://github.com/bitprophet"><img src="https:&#x2F;&#x2F;github.com&#x2F;bitprophet.png" width="60px" alt="User avatar: Jeff Forcier" /></a><a href="https://github.com/tayleighr"><img src="https:&#x2F;&#x2F;github.com&#x2F;tayleighr.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/Novakov"><img src="https:&#x2F;&#x2F;github.com&#x2F;Novakov.png" width="60px" alt="User avatar: Maciej T. Nowak" /></a><a href="https://github.com/jhillyerd"><img src="https:&#x2F;&#x2F;github.com&#x2F;jhillyerd.png" width="60px" alt="User avatar: James Hillyerd" /></a><a href="https://github.com/nekhaevskiy"><img src="https:&#x2F;&#x2F;github.com&#x2F;nekhaevskiy.png" width="60px" alt="User avatar: Yury" /></a><a href="https://github.com/reivilibre"><img src="https:&#x2F;&#x2F;github.com&#x2F;reivilibre.png" width="60px" alt="User avatar: Olivier reivilibre" /></a><a href="https://github.com/BSteffaniak"><img src="https:&#x2F;&#x2F;github.com&#x2F;BSteffaniak.png" width="60px" alt="User avatar: Braden Steffaniak" /></a><a href="https://github.com/jordan-gillard"><img src="https:&#x2F;&#x2F;github.com&#x2F;jordan-gillard.png" width="60px" alt="User avatar: Jordan Gillard" /></a><a href="https://github.com/smangels"><img src="https:&#x2F;&#x2F;github.com&#x2F;smangels.png" width="60px" alt="User avatar: Sebastian" /></a><a href="https://github.com/amslezak"><img src="https:&#x2F;&#x2F;github.com&#x2F;amslezak.png" width="60px" alt="User avatar: Andy Slezak" /></a><a href="https://github.com/mkock"><img src="https:&#x2F;&#x2F;github.com&#x2F;mkock.png" width="60px" alt="User avatar: Martin Kock" /></a><a href="https://github.com/danielkokott"><img src="https:&#x2F;&#x2F;github.com&#x2F;danielkokott.png" width="60px" alt="User avatar: Daniel Kokott" /></a><a href="https://github.com/heijmans"><img src="https:&#x2F;&#x2F;github.com&#x2F;heijmans.png" width="60px" alt="User avatar: Jan Heijmans" /></a><a href="https://github.com/knowald"><img src="https:&#x2F;&#x2F;github.com&#x2F;knowald.png" width="60px" alt="User avatar: Kevin Nowald" /></a><a href="https://github.com/ethanjli"><img src="https:&#x2F;&#x2F;github.com&#x2F;ethanjli.png" width="60px" alt="User avatar: Ethan Li" /></a><a href="https://github.com/phubaba"><img src="https:&#x2F;&#x2F;github.com&#x2F;phubaba.png" width="60px" alt="User avatar: Robert Forler" /></a><a href="https://github.com/neunkasulle"><img src="https:&#x2F;&#x2F;github.com&#x2F;neunkasulle.png" width="60px" alt="User avatar: Jan Zenkner" /></a><a href="https://github.com/RVxLab"><img src="https:&#x2F;&#x2F;github.com&#x2F;RVxLab.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/FrederickGeek8"><img src="https:&#x2F;&#x2F;github.com&#x2F;FrederickGeek8.png" width="60px" alt="User avatar: Frederick Morlock" /></a><a href="https://github.com/ezdac"><img src="https:&#x2F;&#x2F;github.com&#x2F;ezdac.png" width="60px" alt="User avatar: Maximilian Langenfeld" /></a><a href="https://github.com/lppassos"><img src="https:&#x2F;&#x2F;github.com&#x2F;lppassos.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/neilcode"><img src="https:&#x2F;&#x2F;github.com&#x2F;neilcode.png" width="60px" alt="User avatar: Neil Lambert" /></a><a href="https://github.com/dhh"><img src="https:&#x2F;&#x2F;github.com&#x2F;dhh.png" width="60px" alt="User avatar: David Heinemeier Hansson" /></a><a href="https://github.com/ethanfischer"><img src="https:&#x2F;&#x2F;github.com&#x2F;ethanfischer.png" width="60px" alt="User avatar: Ethan Fischer" /></a><a href="https://github.com/poshboytl"><img src="https:&#x2F;&#x2F;github.com&#x2F;poshboytl.png" width="60px" alt="User avatar: Terry Tai" /></a><a href="https://github.com/roesnera"><img src="https:&#x2F;&#x2F;github.com&#x2F;roesnera.png" width="60px" alt="User avatar: Adam Roesner" /></a><a href="https://github.com/seven1m"><img src="https:&#x2F;&#x2F;github.com&#x2F;seven1m.png" width="60px" alt="User avatar: Tim Morgan" /></a><a href="https://github.com/sgoridotla1"><img src="https:&#x2F;&#x2F;github.com&#x2F;sgoridotla1.png" width="60px" alt="User avatar: Maksym Shypulniak" /></a><a href="https://github.com/ADIX7"><img src="https:&#x2F;&#x2F;github.com&#x2F;ADIX7.png" width="60px" alt="User avatar: Kovács Ádám" /></a><a href="https://github.com/slowdub"><img src="https:&#x2F;&#x2F;github.com&#x2F;slowdub.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/serranomorante"><img src="https:&#x2F;&#x2F;github.com&#x2F;serranomorante.png" width="60px" alt="User avatar: Patricio Serrano" /></a><a href="https://github.com/kiriDevs"><img src="https:&#x2F;&#x2F;github.com&#x2F;kiriDevs.png" width="60px" alt="User avatar: Kiri" /></a><a href="https://github.com/StevenMasini"><img src="https:&#x2F;&#x2F;github.com&#x2F;StevenMasini.png" width="60px" alt="User avatar: Steven Masini" /></a><a href="https://github.com/bjornevik"><img src="https:&#x2F;&#x2F;github.com&#x2F;bjornevik.png" width="60px" alt="User avatar: John Even Bjørnevik" /></a><a href="https://github.com/moberst"><img src="https:&#x2F;&#x2F;github.com&#x2F;moberst.png" width="60px" alt="User avatar: Michael Oberst" /></a><a href="https://github.com/adam-e-trepanier"><img src="https:&#x2F;&#x2F;github.com&#x2F;adam-e-trepanier.png" width="60px" alt="User avatar: Adam Trepanier" /></a><a href="https://github.com/arkalon76"><img src="https:&#x2F;&#x2F;github.com&#x2F;arkalon76.png" width="60px" alt="User avatar: Kenth Fagerlund" /></a><a href="https://github.com/Djuuu"><img src="https:&#x2F;&#x2F;github.com&#x2F;Djuuu.png" width="60px" alt="User avatar: Julien Tardot" /></a><a href="https://github.com/ellord"><img src="https:&#x2F;&#x2F;github.com&#x2F;ellord.png" width="60px" alt="User avatar: Ellord Tayag" /></a><a href="https://github.com/EdgarPost"><img src="https:&#x2F;&#x2F;github.com&#x2F;EdgarPost.png" width="60px" alt="User avatar: Edgar Post-Buijs" /></a><a href="https://github.com/caillou"><img src="https:&#x2F;&#x2F;github.com&#x2F;caillou.png" width="60px" alt="User avatar: Pierre Spring" /></a><a href="https://github.com/mebezac"><img src="https:&#x2F;&#x2F;github.com&#x2F;mebezac.png" width="60px" alt="User avatar: Zac Clay" /></a><a href="https://github.com/Tom94"><img src="https:&#x2F;&#x2F;github.com&#x2F;Tom94.png" width="60px" alt="User avatar: Thomas Müller" /></a><a href="https://github.com/ccssmnn"><img src="https:&#x2F;&#x2F;github.com&#x2F;ccssmnn.png" width="60px" alt="User avatar: Carl Assmann" /></a><a href="https://github.com/ognevsd"><img src="https:&#x2F;&#x2F;github.com&#x2F;ognevsd.png" width="60px" alt="User avatar: Sergey Ognev" /></a><a href="https://github.com/moodyhunter"><img src="https:&#x2F;&#x2F;github.com&#x2F;moodyhunter.png" width="60px" alt="User avatar: Moody Liu" /></a><a href="https://github.com/elithper"><img src="https:&#x2F;&#x2F;github.com&#x2F;elithper.png" width="60px" alt="User avatar: Michael Howard" /></a><a href="https://github.com/LasseBloch"><img src="https:&#x2F;&#x2F;github.com&#x2F;LasseBloch.png" width="60px" alt="User avatar: Lasse Bloch Lauritsen" /></a><a href="https://github.com/dbrockman"><img src="https:&#x2F;&#x2F;github.com&#x2F;dbrockman.png" width="60px" alt="User avatar: David Brockman" /></a><a href="https://github.com/slavshik"><img src="https:&#x2F;&#x2F;github.com&#x2F;slavshik.png" width="60px" alt="User avatar: Alexander Slavschik" /></a><a href="https://github.com/aidalgol"><img src="https:&#x2F;&#x2F;github.com&#x2F;aidalgol.png" width="60px" alt="User avatar: Aidan Gauland" /></a><a href="https://github.com/mbienkowsk"><img src="https:&#x2F;&#x2F;github.com&#x2F;mbienkowsk.png" width="60px" alt="User avatar: Maksym Bieńkowski" /></a><a href="https://github.com/joshuawootonn"><img src="https:&#x2F;&#x2F;github.com&#x2F;joshuawootonn.png" width="60px" alt="User avatar: Joshua Wootonn" /></a><a href="https://github.com/I4nJ"><img src="https:&#x2F;&#x2F;github.com&#x2F;I4nJ.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/sandviklee"><img src="https:&#x2F;&#x2F;github.com&#x2F;sandviklee.png" width="60px" alt="User avatar: Simon Sandvik Lee" /></a><a href="https://github.com/glagnar"><img src="https:&#x2F;&#x2F;github.com&#x2F;glagnar.png" width="60px" alt="User avatar: Thomas Gilbert" /></a><a href="https://github.com/skrzepto"><img src="https:&#x2F;&#x2F;github.com&#x2F;skrzepto.png" width="60px" alt="User avatar: Szymon Mucha" /></a><a href="https://github.com/unnawut"><img src="https:&#x2F;&#x2F;github.com&#x2F;unnawut.png" width="60px" alt="User avatar: Unnawut Leepaisalsuwanna" /></a><a href="https://github.com/wortmanb"><img src="https:&#x2F;&#x2F;github.com&#x2F;wortmanb.png" width="60px" alt="User avatar: Bret Wortman" /></a><a href="https://github.com/simoncdna"><img src="https:&#x2F;&#x2F;github.com&#x2F;simoncdna.png" width="60px" alt="User avatar: Simon Cardona" /></a><a href="https://github.com/andre-lameirinhas"><img src="https:&#x2F;&#x2F;github.com&#x2F;andre-lameirinhas.png" width="60px" alt="User avatar: André Lameirinhas" /></a><a href="https://github.com/SVappsLAB"><img src="https:&#x2F;&#x2F;github.com&#x2F;SVappsLAB.png" width="60px" alt="User avatar: Scott Velez" /></a><a href="https://github.com/ooojustin"><img src="https:&#x2F;&#x2F;github.com&#x2F;ooojustin.png" width="60px" alt="User avatar: justin" /></a><a href="https://github.com/mayfieldiv"><img src="https:&#x2F;&#x2F;github.com&#x2F;mayfieldiv.png" width="60px" alt="User avatar: Mayfield" /></a><a href="https://github.com/somaholiday"><img src="https:&#x2F;&#x2F;github.com&#x2F;somaholiday.png" width="60px" alt="User avatar: Soma Holiday" /></a><a href="https://github.com/bizmythy"><img src="https:&#x2F;&#x2F;github.com&#x2F;bizmythy.png" width="60px" alt="User avatar: bizmyth" /></a><a href="https://github.com/dessalines"><img src="https:&#x2F;&#x2F;github.com&#x2F;dessalines.png" width="60px" alt="User avatar: Dessalines" /></a><a href="https://github.com/KKodiac"><img src="https:&#x2F;&#x2F;github.com&#x2F;KKodiac.png" width="60px" alt="User avatar: Sean Hong(홍성민)" /></a><a href="https://github.com/RADreymann"><img src="https:&#x2F;&#x2F;github.com&#x2F;RADreymann.png" width="60px" alt="User avatar: Alex Dreymann" /></a><a href="https://github.com/felipeospina21"><img src="https:&#x2F;&#x2F;github.com&#x2F;felipeospina21.png" width="60px" alt="User avatar: Felipe Ospina" /></a><a href="https://github.com/riccardoNovaglia"><img src="https:&#x2F;&#x2F;github.com&#x2F;riccardoNovaglia.png" width="60px" alt="User avatar: Riccardo Novaglia" /></a><a href="https://github.com/nitipon-apaisri"><img src="https:&#x2F;&#x2F;github.com&#x2F;nitipon-apaisri.png" width="60px" alt="User avatar: rxz" /></a><a href="https://github.com/rbuchberger"><img src="https:&#x2F;&#x2F;github.com&#x2F;rbuchberger.png" width="60px" alt="User avatar: Robert Buchberger" /></a><a href="https://github.com/tyamori"><img src="https:&#x2F;&#x2F;github.com&#x2F;tyamori.png" width="60px" alt="User avatar: housekeeper" /></a><a href="https://github.com/fogrye"><img src="https:&#x2F;&#x2F;github.com&#x2F;fogrye.png" width="60px" alt="User avatar: Yurii" /></a><a href="https://github.com/paalss"><img src="https:&#x2F;&#x2F;github.com&#x2F;paalss.png" width="60px" alt="User avatar: Pål Syvertsen Stakvik" /></a><a href="https://github.com/SaucePackets"><img src="https:&#x2F;&#x2F;github.com&#x2F;SaucePackets.png" width="60px" alt="User avatar: " /></a><a href="https://github.com/zakelfassi"><img src="https:&#x2F;&#x2F;github.com&#x2F;zakelfassi.png" width="60px" alt="User avatar: Zak El Fassi" /></a><!-- sponsors -->
</p>

## Elevator Pitch

Rant time: You've heard it before, git is _powerful_, but what good is that power when everything is so damn hard to do? Interactive rebasing requires you to edit a goddamn TODO file in your editor? _Are you kidding me?_ To stage part of a file you need to use a command line program to step through each hunk and if a hunk can't be split down any further but contains code you don't want to stage, you have to edit an arcane patch file _by hand_? _Are you KIDDING me?!_ Sometimes you get asked to stash your changes when switching branches only to realise that after you switch and unstash that there weren't even any conflicts and it would have been fine to just checkout the branch directly? _YOU HAVE GOT TO BE KIDDING ME!_

If you're a mere mortal like me and you're tired of hearing how powerful git is when in your daily life it's a powerful pain in your ass, lazygit might be for you.

## Features

### Stage individual lines

Press space on the selected line to stage it, or press `v` to start selecting a range of lines. You can also press `a` to select the entirety of the current hunk.

![stage_lines](../assets/demo/stage_lines-compressed.gif)

### Interactive Rebase

Press `i` to start an interactive rebase. Then squash (`s`), fixup (`f`), drop (`d`), edit (`e`), move up (`ctrl+k`) or move down (`ctrl+j`) any of the TODO commits, before continuing the rebase by bringing up the rebase options menu with `m` and then selecting `continue`.

You can also perform any of these actions as a once-off (e.g. pressing `s` on a commit to squash it) without explicitly starting a rebase.

This demo also uses shift+down to select a range of commits to move and fixup.

![interactive_rebase](../assets/demo/interactive_rebase-compressed.gif)

### Cherry-pick

Press `shift+c` on a commit to copy it and press `shift+v` to paste (cherry-pick) it.

![cherry_pick](../assets/demo/cherry_pick-compressed.gif)

### Bisect

Press `b` in the commits view to mark a commit as good/bad in order to begin a git bisect.

![bisect](../assets/demo/bisect-compressed.gif)

### Nuke the working tree

For when you really want to just get rid of anything that shows up when you run `git status` (and yes that includes dirty submodules) [kidpix style](https://www.youtube.com/watch?v=N4E2B_k2Bss), press `shift+d` to bring up the reset options menu and then select the 'nuke' option.

![Nuke working tree](../assets/demo/nuke_working_tree-compressed.gif)

### Amend an old commit

Pressing `shift+a` on any commit will amend that commit with the currently staged changes (running an interactive rebase in the background).

![amend_old_commit](../assets/demo/amend_old_commit-compressed.gif)

### Filter

You can filter a view with `/`. Here we filter down our branches view and then hit `enter` to view its commits.

![filter](../assets/demo/filter-compressed.gif)

### Invoke a custom command

Lazygit has a very flexible [custom command system](docs/Custom_Command_Keybindings.md). In this example a custom command is defined which emulates the built-in branch checkout action.

![custom_command](../assets/demo/custom_command-compressed.gif)

### Worktrees

You can create worktrees to have multiple branches going at once without the need for stashing or creating WIP commits when switching between them. Press `w` in the branches view to create a worktree from the selected branch and switch to it.

![worktree_create_from_branches](../assets/demo/worktree_create_from_branches-compressed.gif)

### Rebase magic (custom patches)

You can build a custom patch from an old commit and then remove the patch from the commit, split out a new commit, apply the patch in reverse to the index, and more.

In this example we have a redundant comment that we want to remove from an old commit. We hit `<enter>` on the commit to view its files, then `<enter>` on a file to focus the patch, then `<space>` to add the comment line to our custom patch, and then `ctrl+p` to view the custom patch options; selecting to remove the patch from the current commit.

Learn more in the [Rebase magic Youtube tutorial](https://youtu.be/4XaToVut_hs).

![custom_patch](../assets/demo/custom_patch-compressed.gif)

### Rebase from marked base commit

Say you're on a feature branch that was itself branched off of the develop branch, and you've decided you'd rather be branching off the master branch. You need a way to rebase only the commits from your feature branch. In this demo we check to see which was the last commit on the develop branch, then press `shift+b` to mark that commit as our base commit, then press `r` on the master branch to rebase onto it, only bringing across the commits from our feature branch. Then we push our changes with `shift+p`.

![rebase_onto](../assets/demo/rebase_onto-compressed.gif)

### Undo

You can undo the last action by pressing `z` and redo with `shift+z`. Here we drop a couple of commits and then undo the actions.
Undo uses the reflog which is specific to commits and branches so we can't undo changes to the working tree or stash.

[More info](/docs/Undoing.md)

![undo](../assets/demo/undo-compressed.gif)

### Commit graph

When viewing the commit graph in an enlarged window (use `+` and `_` to cycle screen modes), the commit graph is shown. Colours correspond to the commit authors, and as you navigate down the graph, the parent commits of the selected commit are highlighted.

![commit_graph](../assets/demo/commit_graph-compressed.gif)

### Compare two commits

If you press `shift+w` on a commit (or branch/ref) a menu will open that allows you to mark that commit so that any other commit you select will be diffed against it. Once you've selected the second commit, you'll see the diff in the main view and if you press `<enter>` you'll see the files of the diff. You can press `shift+w` to view the diff menu again to see options like reversing the diff direction or exiting diff mode. You can also exit diff mode by pressing `<escape>`.

![diff_commits](../assets/demo/diff_commits-compressed.gif)

### Show GitHub pull requests

In the branches panel, lazygit can show which of your branches have an associated GitHub pull request by showing a GitHub icon next to the branch name; its color shows the state of the PR (open, merged, etc.). For those that have one, you can press `shift-G` to open the PR in the browser. There is no configuration needed to enable this for github.com, but it requires the [`gh`](https://cli.github.com/) tool to be installed, and you need to do `gh auth login` once to allow lazygit to access GitHub. For GitHub Enterprise, also run `gh auth login --hostname <webDomain>` and add a [`services` entry](docs/Config.md#custom-pull-request-urls) for the host with the `github` provider.

## Tutorials

[<img src="https://i.imgur.com/sVEktDn.png">](https://youtu.be/CPLdltN7wgE)

- [15 Lazygit Features in 15 Minutes](https://youtu.be/CPLdltN7wgE)
- [Basics Tutorial](https://youtu.be/VDXvbHZYeKY)
- [Rebase Magic Tutorial](https://youtu.be/4XaToVut_hs)

## Installation

[![Packaging status](https://repology.org/badge/vertical-allrepos/lazygit.svg?columns=3)](https://repology.org/project/lazygit/versions)

_Most of the above packages are maintained by third parties so be sure to vet them yourself and confirm that the maintainer is a trustworthy looking person who attends local sports games and gives back to their communities with barbeque fundraisers etc_

### From source

```sh
git clone https://github.com/damiensawyer/lazygit.git
cd lazygit
go install
```

### Homebrew

It works with Linux, too.

```sh
brew install lazygit
```

### Binary Releases

For Windows, Mac OS(10.12+) or Linux, you can download a binary release [here](../../releases).

### Dev container feature

If you want to use lazygit in e.g. one of your GitHub Codespaces, there is a third-party [dev container feature](https://github.com/GeorgOfenbeck/features/tree/main/src/lazygit-linuxbinary) based on the binary releases mentioned above.

### MacPorts

Latest version built from github releases.
Tap:

```
sudo port install lazygit
```

### Void Linux

Packages for Void Linux are available in the distro repo

They follow upstream latest releases

```sh
sudo xbps-install -S lazygit
```

### Scoop (Windows)

You can install `lazygit` using [scoop](https://scoop.sh/). It's in the `extras` bucket:

```sh
# Add the extras bucket
scoop bucket add extras

# Install lazygit
scoop install lazygit
```

### gah (Linux and Mac OS)

You can install `lazygit` using [gah](https://github.com/marverix/gah/):

```sh
gah install lazygit
```

### Arch Linux

Packages for Arch Linux are available via pacman and AUR (Arch User Repository).

There are two packages. The stable one which is built with the latest release
and the git version which builds from the most recent commit.

- Stable: `sudo pacman -S lazygit`
- Development: <https://aur.archlinux.org/packages/lazygit-git/>

Instruction of how to install AUR content can be found here:
<https://wiki.archlinux.org/index.php/Arch_User_Repository>

### Fedora / Amazon Linux 2023 / CentOS Stream

Packages for Fedora, Amazon Linux 2023 and CentOS Stream are available via
[Copr](https://copr.fedorainfracloud.org/coprs/dejan/lazygit/) (Cool Other Package Repo).

```sh
sudo dnf copr enable dejan/lazygit
sudo dnf install lazygit
```

These packages are built using the RPM spec file located here: https://codeberg.org/dejan/rpm-lazygit

You should be able to build RPMs for Fedora 41 or older, and other Fedora derivatives using the
SRPM (Source RPM) file that you can grab from the latest COPR build.

#### Fedora / RHEL Derivatives (Terra)

Packages for Fedora and RHEL derivatives are also available from the [Terra Repository](https://terra.fyralabs.com/).

```sh
sudo dnf install --nogpgcheck --repofrompath 'terra,https://repos.fyralabs.com/terra$releasever' terra-release
sudo dnf install lazygit
```

(Install guide for Atomic/EL can be found on their [README](https://github.com/terrapkg/packages/pull/9747))

Terra also has `lazygit-doc`, which contains the contents of the [docs](https://github.com/jesseduffield/lazygit/tree/master/docs) folder.

### Solus Linux

```sh
sudo eopkg install lazygit
```

### Debian and Ubuntu

For **Debian 13 "Trixie", Sid**, and later, or **Ubuntu 25.10 "Questing Quokka"** and later:

```sh
sudo apt install lazygit
```

For **Debian 12 "Bookworm", Ubuntu 25.04 "Plucky Puffin"** and earlier:

```sh
LAZYGIT_VERSION=$(curl -s "https://api.github.com/repos/jesseduffield/lazygit/releases/latest" | \grep -Po '"tag_name": *"v\K[^"]*')
LAZYGIT_ARCH=$(uname -m | sed -e 's/aarch64/arm64/')
curl -Lo lazygit.tar.gz "https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_Linux_${LAZYGIT_ARCH}.tar.gz"
tar xf lazygit.tar.gz lazygit
sudo install lazygit -D -t /usr/local/bin/
```

Verify the correct installation of lazygit:

```sh
lazygit --version
```

### Funtoo Linux

Funtoo Linux has an autogenerated lazygit package in [dev-kit](https://github.com/funtoo/dev-kit/tree/1.4-release/dev-vcs/lazygit):

```sh
sudo emerge dev-vcs/lazygit
```

### Gentoo Linux

Lazygit is not (yet) in main Gentoo portage, however an ebuild is available in [GURU overlay](https://github.com/gentoo-mirror/guru/tree/master/dev-vcs/lazygit)

You can either add the overlay to your system and install lazygit as usual:

```sh
sudo eselect repository enable guru
sudo emaint sync -r guru
sudo emerge dev-vcs/lazygit
```

### openSUSE

The lazygit package is currently built in [devel:languages:go/lazygit](https://build.opensuse.org/package/show/devel:languages:go/lazygit).

To install lazygit on openSUSE Tumbleweed run:

```sh
sudo zypper ar https://download.opensuse.org/repositories/devel:/languages:/go/openSUSE_Factory/devel:languages:go.repo
sudo zypper ref && sudo zypper in lazygit
```

To install lazygit on openSUSE Leap run:

```sh
source /etc/os-release
sudo zypper ar https://download.opensuse.org/repositories/devel:/languages:/go/$VERSION_ID/devel:languages:go.repo
sudo zypper ref && sudo zypper in lazygit
```

### NixOS

#### Using lazygit from nixpkgs

On NixOS, lazygit is packaged with nix and distributed via nixpkgs.
You can try lazygit without installing it with:

```sh
nix-shell -p lazygit
# or with flakes enabled
nix run nixpkgs#lazygit
```

Or you can add lazygit to your `configuration.nix` using the `environment.systemPackages` option.
More details can be found via NixOS search [page](https://search.nixos.org/).

#### Using the official lazygit flake

This repository includes a nix flake that provides the latest development version and additional development tools:

**Run lazygit directly from the repository:**

```sh
nix run github:jesseduffield/lazygit
# or from a local clone
nix run .
```

**Build lazygit from source:**

```sh
nix build github:jesseduffield/lazygit
# or from a local clone
nix build .
```

**Development environment:**
For contributors, the flake provides a development shell with Go toolchain, development tools, and dependencies:

```sh
nix develop github:jesseduffield/lazygit
# or from a local clone
nix develop
```

The development shell includes:

- Go toolchain
- git and make
- Proper environment variables for development

**Using in other flakes:**
The flake also provides an overlay for easy integration into other flake-based projects:

```nix
{
  inputs.lazygit.url = "github:jesseduffield/lazygit";

  outputs = { self, nixpkgs, lazygit }: {
    # Use the overlay
    nixpkgs.overlays = [ lazygit.overlays.default ];
  };
}
```

### Flox

Lazygit can be installed into a Flox environment as follows.

```sh
flox install lazygit
```

More details about Flox can be found on [their website](https://flox.dev/).

### FreeBSD

```sh
pkg install lazygit
```

### Termux

```sh
apt install lazygit
```

### Conda

Released versions are available for different platforms, see <https://anaconda.org/conda-forge/lazygit>

```sh
conda install -c conda-forge/lazygit
```

### Go

```sh
go install github.com/jesseduffield/lazygit@latest
```

Please note:
If you get an error claiming that lazygit cannot be found or is not defined, you
may need to add `~/go/bin` to your $PATH (macOS/Linux), or `%HOME%\go\bin`
(Windows). Not to be mistaken for `C:\Go\bin` (which is for Go's own binaries,
not apps like lazygit).

### Chocolatey (Windows)

You can install `lazygit` using [Chocolatey](https://chocolatey.org/):

```sh
choco install lazygit
```

### Winget (Windows 10 1709 or later)

You can install `lazygit` using the `winget` command in the Windows Terminal with the following command:

```powershell
winget install -e --id=JesseDuffield.lazygit
```

### Manual

You'll need to [install Go](https://golang.org/doc/install)

```
git clone https://github.com/jesseduffield/lazygit.git
cd lazygit
go install
```

You can also use `go run main.go` to compile and run in one go (pun definitely intended)

## Usage

Call `lazygit` in your terminal inside a git repository.

```sh
$ lazygit
```

If you want, you can
also add an alias for this with `echo "alias lg='lazygit'" >> ~/.zshrc` (or
whichever rc file you're using).

### Keybindings

You can check out the list of keybindings [here](/docs/keybindings).

### Changing Directory On Exit

If you change repos in lazygit and want your shell to change directory into that repo on exiting lazygit, add this to your `~/.zshrc` (or other rc file):

```
lg()
{
    export LAZYGIT_NEW_DIR_FILE=~/.lazygit/newdir

    lazygit "$@"

    if [ -f $LAZYGIT_NEW_DIR_FILE ]; then
            cd "$(cat $LAZYGIT_NEW_DIR_FILE)"
            rm -f $LAZYGIT_NEW_DIR_FILE > /dev/null
    fi
}
```

Then `source ~/.zshrc` and from now on when you call `lg` and exit you'll switch directories to whatever you were in inside lazygit. To override this behaviour you can exit using `shift+Q` rather than just `q`.

### Undo/Redo

See the [docs](/docs/Undoing.md)

## Configuration

Check out the [configuration docs](docs/Config.md).

### Custom Diff Renderers

See the [docs](docs/Custom_DiffRenderers.md)

### Custom Commands

If lazygit is missing a feature, there's a good chance you can implement it yourself with a custom command!

See the [docs](docs/Custom_Command_Keybindings.md)

### Git flow support

Lazygit supports [Gitflow](https://github.com/nvie/gitflow) (or [git-flow-next](https://github.com/gittower/git-flow-next)) if you have it installed. To understand how the Gitflow model works check out Vincent Driessen's original [post](https://nvie.com/posts/a-successful-git-branching-model/) explaining it. To view Gitflow options from within Lazygit, press `i` from within the branches view.

## Contributing

We love your input! Please check out the [contributing guide](CONTRIBUTING.md).
For contributor discussion about things not better discussed here in the repo, join the [discord channel](https://discord.gg/ehwFt2t4wt)

<a href="https://discord.gg/ehwFt2t4wt"><img src='../assets/discord.png' width='75'></a>

Check out this [video](https://www.youtube.com/watch?v=kNavnhzZHtk) walking through the creation of a small feature in lazygit if you want an idea of where to get started.

### Debugging Locally

Run `lazygit --debug` in one terminal tab and `lazygit --logs` in another to view the program and its log output side by side

## Donate

If you would like to support the development of lazygit, consider [sponsoring me](https://github.com/sponsors/jesseduffield) (github is matching all donations dollar-for-dollar for 12 months)

## FAQ

### What do the commit colors represent?

- Green: the commit is included in the master branch
- Yellow: the commit is not included in the master branch
- Red: the commit has not been pushed to the upstream branch

## Shameless Plug

If you want to see what I (Jesse) am up to in terms of development, follow me on
[twitter](https://twitter.com/DuffieldJesse) or check out my [blog](https://jesseduffield.com/)

## Alternatives

If you find that lazygit doesn't quite satisfy your requirements, these may be a better fit:

- [GitUI](https://github.com/Extrawurst/gitui)
- [tig](https://github.com/jonas/tig)
- [GitArbor TUI](https://github.com/cadamsdev/gitarbor-tui)
