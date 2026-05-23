def main() -> None:
    d: dict[str, int] = {"b": 2, "a": 1, "c": 3}
    sorted_keys: list[str] = sorted(d.keys())
    for k in sorted_keys:
        print(k, d[k])
    doubled: dict[str, int] = {}
    for k in sorted(d.keys()):
        doubled[k] = d[k] * 2
    for k in sorted(doubled.keys()):
        print(k, doubled[k])
    del d["b"]
    for k in sorted(d.keys()):
        print(k, d[k])


if __name__ == "__main__":
    main()
