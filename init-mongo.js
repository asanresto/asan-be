db.createUser({
  user: "mongo",
  pwd: "password123",
  roles: [
    {
      role: "readWrite",
      db: "asan",
    },
  ],
});
