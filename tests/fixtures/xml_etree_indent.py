import xml.etree.ElementTree as ET


def main() -> None:
    root = ET.Element("root")
    a = ET.SubElement(root, "a")
    a.text = "x"
    b = ET.SubElement(root, "b")
    b.text = "y"
    ET.indent(root, "  ")
    print(ET.tostring(root, encoding="unicode"))


if __name__ == "__main__":
    main()
