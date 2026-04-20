import hashlib
import itertools


def md5_hash(text):
    return hashlib.md5(text.encode()).hexdigest()


def load_hashes(file):
    data = []
    with open(file, "r") as f:
        for line in f:
            if "," in line:
                email, h = line.strip().split(",", 1)
                data.append((email.strip(), h.strip()))
    return data


def generate_passwords():
    words = ["milo", "melbourne"]
    numbers = ["123", "1988"]

    candidates = []

    # 🔥 generate permutations of words
    for r in range(1, 3):  # 1-word and 2-word combos
        for combo in itertools.permutations(words, r):
            base = "".join(combo)

            for num in numbers:
                # pattern: words + number + @
                candidates.append(base + num + "@")

                # also try number before words
                candidates.append(num + base + "@")

    print(f"Generated {len(candidates)} candidates")
    return candidates


def crack(password_dump):
    candidates = generate_passwords()

    for pwd in candidates:
        h = md5_hash(pwd)

        for email, hash_value in password_dump:
            if h == hash_value:
                print("\n✅ MATCH FOUND")
                print("Email:", email)
                print("Password:", pwd)
                print("MD5:", h)
                return

    print("\n❌ No match found")


if __name__ == "__main__":
    dump = load_hashes("password_dump.txt")
    crack(dump)