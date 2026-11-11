"""Deterministic Latin preview subsets of the typeface plugins' full fonts (ADR-0107 section 3).

  font-previews.py generate <plugins-dir> <out-dir> [plugin...]
  font-previews.py rfnscan  <preview.woff2> <rfn> [<rfn>...]

Run through scripts/subset-previews.sh, which supplies the pinned fontTools. The output depends
only on the plugin's full font, scripts/font-previews.json and the pinned tools: the Unicode set
and layout features are fixed, hinting is dropped, head timestamps are fixed and a family with a
Reserved Font Name is renamed (an OFL Modified Version may not keep the name).
"""
import hashlib, json, os, re, sys

from fontTools import subset
from fontTools.ttLib import TTFont

HERE = os.path.dirname(os.path.abspath(__file__))
# name IDs that are legal text or version strings and may legitimately carry an RFN: copyright (0),
# version (5), trademark (7), manufacturer (8), designer (9), vendor and designer URLs (11, 12),
# licence text, URL (13, 14).
# 2026-01-01T00:00:00Z in the 1904 epoch fontTools uses
FIXED_TIME = 3850070400
LEGAL_NAME_IDS = {0, 5, 7, 8, 9, 11, 12, 13, 14}


def sha256(path):
    with open(path, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def slug(s):
    return re.sub(r"[^a-z0-9]+", "-", s.lower()).strip("-")


def swap(text, old, new):
    # family name as written, and its space-free form as used in PostScript names
    text = text.replace(old, new).replace(old.replace(" ", ""), new.replace(" ", ""))
    return text


def rename(font, old, new):
    for rec in font["name"].names:
        if rec.nameID in LEGAL_NAME_IDS:
            continue
        rec.string = swap(rec.toUnicode(), old, new)
    if "CFF " in font:
        cff = font["CFF "].cff
        cff.fontNames = [swap(n, old, new) for n in cff.fontNames]
        top = cff.topDictIndex[0]
        for attr in ("FullName", "FamilyName"):
            if hasattr(top, attr):
                setattr(top, attr, swap(getattr(top, attr), old, new))


def name_leaks(font, rfns):
    out = []
    for rec in font["name"].names:
        if rec.nameID in LEGAL_NAME_IDS:
            continue
        s = rec.toUnicode()
        out += [(rec.nameID, r) for r in rfns if r.lower() in s.lower()]
    if "CFF " in font:
        cff = font["CFF "].cff
        top = cff.topDictIndex[0]
        strings = list(cff.fontNames) + [getattr(top, a, "") for a in ("FullName", "FamilyName")]
        out += [("CFF", r) for s in strings for r in rfns if r.lower() in s.lower()]
    return out


def cut(src, dst, cfg, family_rename):
    args = [
        src,
        "--output-file=" + dst,
        "--flavor=woff2",
        "--unicodes=" + cfg["unicodes"],
        "--layout-features=" + cfg["layout_features"],
        "--no-hinting",
        "--desubroutinize",
        "--no-glyph-names",
        "--notdef-outline",
        "--no-recalc-timestamp",
    ]
    if family_rename:
        # the name table is rewritten below, so keep every record the subsetter would drop
        args.append("--name-IDs=*")
    subset.main(args)
    font = TTFont(dst, recalcTimestamp=False)
    font.flavor = None
    font["head"].created = FIXED_TIME
    font["head"].modified = FIXED_TIME
    if family_rename:
        rename(font, *family_rename)
    font.flavor = "woff2"
    font.save(dst)


def generate(plugins, out, names):
    cfg = json.load(open(os.path.join(HERE, "font-previews.json")))
    if not names:
        names = sorted(
            d for d in os.listdir(plugins)
            if os.path.isfile(os.path.join(plugins, d, ".claude-plugin", "plugin.json"))
            and json.load(open(os.path.join(plugins, d, ".claude-plugin", "plugin.json"))).get("fonts")
        )
    for name in names:
        pdir = os.path.join(plugins, name)
        man = json.load(open(os.path.join(pdir, ".claude-plugin", "plugin.json")))
        fams = {}
        for e in man["fonts"]:
            fams.setdefault(e["family"], {}).setdefault(e["file"], e)
        for family, files in sorted(fams.items()):
            first = next(iter(files.values()))
            rfn = first["rfn"]
            rec = {"choice": "none"} if rfn == "none" else None
            new = None
            if rfn != "none":
                new = cfg["rfn_renames"].get(family)
                if not new:
                    sys.exit(f"{name}: {family} declares RFN {rfn!r} and has no rfn_renames entry")
                if any(r.strip().lower() in new.lower() for r in rfn.split(",")):
                    sys.exit(f"{name}: rename {new!r} keeps RFN {rfn!r}")
                rec = {"choice": "renamed", "preview_family": new}
            # paths are named from the preview family: a renamed family's files must not keep the RFN
            base = slug(new or family)
            fdir = os.path.join(out, name, base)
            os.makedirs(fdir, exist_ok=True)
            entries = []
            for rel, e in sorted(files.items()):
                src = os.path.join(pdir, rel)
                if sha256(src) != e["sha256"]:
                    sys.exit(f"{name}: {rel} does not match its recorded sha256")
                prev = f"{base}-{e['weight'].replace(' ', '-')}-{e['style']}.woff2"
                dst = os.path.join(fdir, prev)
                cut(src, dst, cfg, (family, new) if new else None)
                if new:
                    f = TTFont(dst)
                    leaks = name_leaks(f, [r.strip() for r in rfn.split(",")])
                    if leaks:
                        sys.exit(f"{name}: {prev} keeps RFN in {leaks}")
                entries.append({
                    "weight": e["weight"], "style": e["style"], "full_file": rel,
                    "full_sha256": e["sha256"], "preview": prev, "preview_sha256": sha256(dst),
                    "preview_bytes": os.path.getsize(dst),
                })
            doc = {
                "plugin": name, "plugin_version": man["version"], "family": family,
                "source": first["source"], "rfn": rfn, "rfn_choice": rec,
                "unicodes": cfg["unicodes"], "files": entries,
            }
            with open(os.path.join(fdir, "manifest.json"), "w") as f:
                json.dump(doc, f, indent=2)
                f.write("\n")


def rfnscan(path, rfns):
    leaks = name_leaks(TTFont(path), rfns)
    for l in leaks:
        print(f"{path}: name {l[0]} keeps Reserved Font Name {l[1]!r}", file=sys.stderr)
    sys.exit(1 if leaks else 0)


if __name__ == "__main__":
    if sys.argv[1] == "generate":
        generate(sys.argv[2], sys.argv[3], sys.argv[4:])
    elif sys.argv[1] == "rfnscan":
        rfnscan(sys.argv[2], sys.argv[3:])
    else:
        sys.exit(__doc__)
