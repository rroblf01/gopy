import bz2
import os
import tempfile
import subprocess
import shutil


def main() -> None:
    bz_bin = shutil.which("bzip2")
    if not bz_bin:
        print("hello")
        return
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("msg.txt", "w") as fh:
        fh.write("hello\n")
    subprocess.run(["bzip2", "-z", "-f", "msg.txt"])

    body: str = ""
    with bz2.open("msg.txt.bz2", "rt") as fh:
        body = fh.read()
    print(body.strip())


if __name__ == "__main__":
    main()
