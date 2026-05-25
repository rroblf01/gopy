import xml.etree.ElementTree as ET


def main() -> None:
    root = ET.fromstring("<r><a>1</a><b>2</b><c>3</c></r>")
    parts = list(root.itertext())
    print(",".join(parts))


if __name__ == "__main__":
    main()
