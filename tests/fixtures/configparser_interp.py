import configparser
import os
import tempfile


def main() -> None:
    base = tempfile.mkdtemp(prefix="gopy_cfg_")
    path = os.path.join(base, "app.ini")

    with open(path, "w") as fh:
        fh.write(
            "[DEFAULT]\n"
            "host = example.com\n"
            "port = 80\n"
            "url = http://%(host)s:%(port)s/api\n"
            "\n"
            "[prod]\n"
            "port = 443\n"
            "url = https://%(host)s:%(port)s/api\n"
            "[debug]\n"
            "enabled = true\n"
            "retries = 3\n"
            "factor = 1.25\n"
        )

    cp = configparser.ConfigParser()
    cp.read(path)

    print(cp.get("prod", "url"))
    print(cp.get("DEFAULT", "url"))
    print(cp.getint("debug", "retries"))
    print(cp.getfloat("debug", "factor"))
    print(cp.getboolean("debug", "enabled"))


if __name__ == "__main__":
    main()
