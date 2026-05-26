def main() -> None:
    with open("/tmp/gopy_mw_a.txt", "w") as a, open("/tmp/gopy_mw_b.txt", "w") as b:
        a.write("alpha")
        b.write("beta")

    with open("/tmp/gopy_mw_a.txt", "r") as a, open("/tmp/gopy_mw_b.txt", "r") as b:
        print(a.read())
        print(b.read())


if __name__ == "__main__":
    main()
