from string import Template


def main() -> None:
    t = Template("$name is $age")
    print(t.substitute({"name": "ada", "age": 36}))
    t2 = Template("${greeting}, ${name}!")
    print(t2.substitute({"greeting": "hi", "name": "bob"}))
    t3 = Template("$a and $b")
    print(t3.safe_substitute({"a": "x"}))


if __name__ == "__main__":
    main()
