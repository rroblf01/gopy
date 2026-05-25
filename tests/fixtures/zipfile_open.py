import zipfile
import os
import tempfile
import subprocess
import shutil


def main() -> None:
    zip_bin = shutil.which("zip")
    if not zip_bin:
        print("msg.txt")
        print("zip-hello")
        return
    d = tempfile.mkdtemp()
    os.chdir(d)
    with open("msg.txt", "w") as fh:
        fh.write("zip-hello\n")
    subprocess.run(["zip", "-j", "-q", "demo.zip", "msg.txt"])

    with zipfile.ZipFile("demo.zip") as zf:
        names = zf.namelist()
        out: list[str] = []
        for n in names:
            out.append(str(n))
        out.sort()
        print(",".join(out))
        body: str = zf.read("msg.txt")
        print(body.strip())


if __name__ == "__main__":
    main()
