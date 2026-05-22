from pathlib import Path


def main() -> None:
    p = Path("/tmp/gopy_test_demo.txt")
    p.write_text("hello world\n")
    print(p.exists())
    print(p.is_file())
    content: str = p.read_text()
    print(content.strip())
    print(p.name)
    print(p.suffix)


if __name__ == "__main__":
    main()
