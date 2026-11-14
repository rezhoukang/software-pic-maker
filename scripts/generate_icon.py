"""Convert the supplied EZ logo to Wails PNG and Windows ICO assets (Pillow)."""
from pathlib import Path
from PIL import Image

root = Path(__file__).resolve().parents[1]
source = Image.open(root/'frontend/src/assets/logo-master.png').convert('RGBA')
source.thumbnail((512, 512), Image.Resampling.LANCZOS)
im = Image.new('RGBA', (512, 512), (0, 0, 0, 0))
im.alpha_composite(source, ((512-source.width)//2, (512-source.height)//2))
(root/'build/windows').mkdir(parents=True, exist_ok=True)
im.save(root/'build/appicon.png')
im.save(root/'build/windows/icon.ico', sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)])
