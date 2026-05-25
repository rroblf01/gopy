import os
import tempfile
import xml.etree.ElementTree as ET


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_xml_")
    src = os.path.join(base, "in.xml")
    dst = os.path.join(base, "out.xml")

    with open(src, "w") as fh:
        fh.write("<root><a v='1'/><b/></root>")

    tree = ET.parse(src)
    root = tree.getroot()
    print(root.tag)
    print(root.find("a").get("v"))

    # Mutate then write.
    new = ET.SubElement(root, "c")
    new.set("k", "v")
    tree.write(dst)

    body: str = ""
    with open(dst) as fh:
        body = fh.read()
    # Strip XML declaration so CPython (which writes it differently) and
    # gopy (which always writes a fixed UTF-8 prolog) align on the body.
    if body.startswith("<?xml"):
        idx: int = body.index("?>")
        body = body[idx + 2:]
    print(body)


if __name__ == "__main__":
    main()
