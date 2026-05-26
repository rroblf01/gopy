import shutil
import os
import tempfile


def main() -> None:
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("src.txt", "w") as fh:
        fh.write("hello world\n")
    with open("src.txt", "r") as src:
        with open("dst.txt", "w") as dst:
            shutil.copyfileobj(src, dst)
    body: str = ""
    with open("dst.txt", "r") as f:
        body = f.read()
    print(body.strip())


if __name__ == "__main__":
    main()
