from http.cookies import SimpleCookie


def main() -> None:
    c = SimpleCookie()
    c["session"] = "abc123"
    c["pref"] = "dark"
    print(len(c))
    print("session" in c)
    print("missing" in c)
    keys = c.keys()
    out: list[str] = []
    for k in keys:
        out.append(str(k))
    out.sort()
    print(",".join(out))


if __name__ == "__main__":
    main()
