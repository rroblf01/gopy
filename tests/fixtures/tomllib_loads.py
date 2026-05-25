import tomllib


def main() -> None:
    src = """
# comment
title = "TOML Example"
count = 42
ratio = 3.14
enabled = true

[server]
host = "localhost"
port = 8080
"""
    doc = tomllib.loads(src)
    print(doc["title"])
    print(doc["count"])
    print(doc["enabled"])
    # gopy: doc["server"] is `any`; CPython gives nested dict.
    # Both should at least confirm the key exists.
    print("server" in doc)


if __name__ == "__main__":
    main()
