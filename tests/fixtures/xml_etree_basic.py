import xml.etree.ElementTree as ET


def main() -> None:
    xml = "<root><item id='1'>alpha</item><item id='2'>beta</item></root>"
    root = ET.fromstring(xml)
    print(root.tag)
    print(len(root.findall("item")))
    for it in root.findall("item"):
        print(it.text)
        print(it.get("id"))


if __name__ == "__main__":
    main()
