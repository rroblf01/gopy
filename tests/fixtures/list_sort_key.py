def main() -> None:
    ws: list[str] = ["alpha", "no", "bbb", "z", "delta"]
    ws.sort(key=lambda s: len(s))
    for s in ws:
        print(s)
    print("---")
    ns: list[int] = [3, -7, 2, -10, 5]
    ns.sort(key=lambda n: -n)
    for n in ns:
        print(n)


if __name__ == "__main__":
    main()
