def main() -> None:
    s = "abcabcabc"
    print(s.find("b"))
    print(s.rfind("b"))
    print(s.find("z"))
    print(s.find("b", 3))
    print(s.find("b", 3, 6))
    print(s.find("b", 100))
    print(s.rfind("b", 0, 5))
    print(s.rfind("z", 0, 5))
    csv = "a,b,c,d,e"
    print(csv.rsplit(",", 1))
    print(csv.rsplit(",", 2))
    print(csv.rsplit(",", 0))
    print(csv.rsplit(","))


if __name__ == "__main__":
    main()
