# Generators & Creation Tools

## Animation Editors
| Tool | Platform | Features |
|---|---|---|
| **durdraw** | Terminal | Frame-by-frame editor, 256 colors, Unicode, CP437, mouse, themes. https://durdraw.org |
| **ASCII Motion** | Browser | Canvas editor with brush/eraser/bezier/text/gradient, export for CLI/web. https://ascii-motion.com |
| **Playscii** | Desktop | Open-source ASCII/ANSI art + animation editor. http://vectorpoem.com/playscii/ |
| **PabloDraw** | Desktop | ANSI/ASCII editor with real-time collaboration. http://picoe.ca/products/pablodraw/ |

## Image → ASCII Converters
| Tool | Notes |
|---|---|
| **chafa** | Best-in-class. Truecolor, auto-detects terminal capabilities. `brew install chafa` |
| **jp2a** | JPEG → ASCII. https://github.com/cslarsen/jp2a |
| **libcaca** | Color ASCII library, converts images/video. http://caca.zoy.org/wiki/libcaca |
| **AAlib** | Portable ASCII art graphics library (classic). http://aa-project.sourceforge.net/ |
| **ascii.life** | AI-powered. 12 retro themes (Matrix, CRT, C64, DOS, Apple II, Amber, Phosphor, Plasma, VFD, LCD). Export GIF/MP4. https://ascii.life |

## GIF → Terminal Animation
| Tool | Language |
|---|---|
| **gif-for-cli** | Go. Purpose-built. |
| **gif2cli** | Python |
| **DIY pipeline** | `ffmpeg -i anim.gif frame_%03d.jpg && chafa --colors 256 frame_*.jpg` |

## Video → ASCII
| Tool | Notes |
|---|---|
| **video-to-ascii** | Python. `pip install video-to-ascii` |
| **mplayer -vo aa/caca** | Play any video as ASCII in terminal |
| **Carbonyl** | Full Chromium browser rendered in ASCII. https://github.com/fathyb/carbonyl |
| **DIY pipeline** | `ffmpeg -i video.mp4 -r 10 frame_%04d.jpg && chafa frame_*.jpg` |

## Text Banner Generators
| Tool | Notes |
|---|---|
| **figlet** | 300+ fonts. `brew install figlet` |
| **toilet** | Color support, filters. `brew install toilet` |
| **lolcat** | Rainbow coloring. `brew install lolcat` |
| **boxes** | Text boxing. `brew install boxes` |
| **neofetch/fastfetch** | System info with ASCII logos |

## Declarative Animation
- **VHS** (Charm) — Write terminal demos as tape files. `brew install vhs`
  https://github.com/charmbracelet/vhs

## Color/Styling Utilities
- **chalk** (Node) — Terminal string styling
- **rich** (Python) — Terminal formatting
- **Lip Gloss** (Go) — Terminal styling
- **tput** — POSIX standard for terminal capabilities
