import xml.etree.ElementTree as ET


def main() -> None:
    src = "<root><a v='1'/><b/></root>"
    root = ET.fromstring(src)

    a = root.find("a")
    a.set("v", "99")
    a.set("name", "alpha")

    c = ET.Element("c")
    c.set("z", "9")
    root.append(c)

    d = ET.SubElement(root, "d")
    d.set("key", "val")

    out = ET.tostring(root, encoding="unicode")
    print(out)


if __name__ == "__main__":
    main()
