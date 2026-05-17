class Greeter:
    def hello(self) -> str:
        return "hello"


class Shouter:
    def shout(self, msg: str) -> str:
        return msg + "!"


class Loud(Greeter, Shouter):
    def __init__(self, name: str) -> None:
        self.name = name

    def announce(self) -> str:
        return self.shout(self.hello() + " " + self.name)


def main() -> None:
    g: Loud = Loud("ada")
    print(g.hello())
    print(g.shout("yo"))
    print(g.announce())
    print(g.name)


if __name__ == "__main__":
    main()
