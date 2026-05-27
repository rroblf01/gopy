def greet(name: str) -> str:
    return "hi " + name


total = 0
for i in range(3):
    total += i
print("total", total)

names = ["a", "b", "c"]
for n in names:
    print(greet(n))

if total > 2:
    print("big")
else:
    print("small")

acc = []
for n in names:
    acc.append(n.upper())
print(acc)
