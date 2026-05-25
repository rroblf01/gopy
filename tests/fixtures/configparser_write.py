import configparser
import os
import tempfile


def main() -> None:
    cp = configparser.ConfigParser()
    cp.add_section("server")
    cp.set("server", "host", "example.com")
    cp.set("server", "port", "8080")
    path: str = os.path.join(tempfile.gettempdir(), "gopy_cp_write.ini")
    with open(path, "w") as fh:
        cp.write(fh)
    # Round-trip
    cp2 = configparser.ConfigParser()
    cp2.read(path)
    print(cp2.get("server", "host"))
    print(cp2.get("server", "port"))
    os.remove(path)


if __name__ == "__main__":
    main()
