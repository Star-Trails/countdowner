# /// script
# requires-python = ">=3.11"
# dependencies = ["fonttools==4.61.1"]
# ///
"""Regenerate the UI fonts with `uv run scripts/subset-ui-fonts.py`."""

import json
from pathlib import Path
from tempfile import TemporaryDirectory
from urllib.request import urlretrieve

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "resources" / "fonts"
SOURCE = "https://raw.githubusercontent.com/notofonts/noto-cjk/523d033d6cb47f4a80c58a35753646f5c3608a78"
REGIONS = {
    "en": ("Latin", "JP", ""),
    "zh-Hant": ("TC", "TC", "正體中文月日一二三四五六"),
    "zh-Hans": ("SC", "SC", "简体中文月日一二三四五六"),
    "ja": ("JP", "JP", "日本語月火水木金土"),
    "ko": ("KR", "KR", "한국어월일화수목금토"),
}


def subset_font(source, locale, region, extra):
    catalog = json.loads((ROOT / "locales" / f"{locale}.json").read_text(encoding="utf-8"))
    translations = "".join(value for value in catalog.values() if isinstance(value, str))
    characters = set(translations + extra + "—−‹›")
    characters.update(map(chr, range(32, 127)))
    characters = {ord(char) for char in characters if char.isprintable()}
    font = TTFont(source, recalcTimestamp=False)
    missing = characters - font.getBestCmap().keys()
    if missing:
        raise ValueError(f"{locale}: missing glyphs for {''.join(map(chr, sorted(missing)))}")

    worker = subset.Subsetter()
    worker.populate(unicodes=characters)
    worker.subset(font)
    font = instantiateVariableFont(font, {"wght": (400, 400, 700)}, inplace=True)

    names = font["name"]
    family = f"Countdowner UI {region}"
    postscript_name = f"CountdownerUI-{region}"
    for record in names.names:
        name = {
            1: family,
            3: postscript_name,
            4: family,
            6: postscript_name,
            16: family,
            25: postscript_name,
        }.get(record.nameID)
        if name is not None:
            record.string = name.encode(record.getEncoding())
    font.save(OUTPUT / f"{region}.ttf")


def main():
    OUTPUT.mkdir(parents=True, exist_ok=True)
    with TemporaryDirectory() as temporary:
        sources = {}
        for locale, (region, source_region, extra) in REGIONS.items():
            if source_region not in sources:
                source = Path(temporary) / f"{source_region}.ttf"
                urlretrieve(f"{SOURCE}/Sans/Variable/TTF/Subset/NotoSans{source_region}-VF.ttf", source)
                sources[source_region] = source
            subset_font(sources[source_region], locale, region, extra)

    license_path = OUTPUT / "OFL.txt"
    urlretrieve(f"{SOURCE}/LICENSE", license_path)
    license_text = license_path.read_text(encoding="utf-8")
    license_path.write_text(
        "Copyright 2014–2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'.\n"
        "Countdowner UI contains renamed subsets of Noto Sans CJK Sans2.004.\n\n" + license_text,
        encoding="utf-8",
    )


if __name__ == "__main__":
    main()
