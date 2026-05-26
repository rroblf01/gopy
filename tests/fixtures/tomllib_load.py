import tomllib
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("conf.toml", "w") as fh:
        fh.write("title = \"hello\"\ncount = 7\n")
    doc: dict[str, any] = {}
    with open("conf.toml", "rb") as fh:
        doc = tomllib.load(fh)
    print(doc["title"])
    print(doc["count"])


if __name__ == "__main__":
    main()
