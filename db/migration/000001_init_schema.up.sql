CREATE TABLE "corrpairs" (
  "date" varchar NOT NULL,
  "x0" varchar NOT NULL,
  "x1" varchar NOT NULL,
  "corr" float(53) NOT NULL
);
CREATE TABLE "historicaldata" (
  "date" varchar NOT NULL,
  "ticker" varchar NOT NULL,
  "k" float(53) NOT NULL,
  "t" float(53) NOT NULL,
  "ivol" float(53) NOT NULL,
  "underlying" varchar NOT NULL
);
CREATE TABLE "modelparameters" (
  "date" varchar NOT NULL,
  "ticker" varchar NOT NULL,
  "sigma" float(53) NOT NULL,
  "alpha" float(53) NOT NULL,
  "beta" float(53) NOT NULL,
  "kappa" float(53) NOT NULL,
  "rho" float(53) NOT NULL
);
CREATE TABLE "statistics" (
  "date" varchar NOT NULL,
  "ticker" varchar NOT NULL,
  "index" integer NOT NULL,
  "mean" float(53) NOT NULL,
  "fixing" float(53) NOT NULL
);
CREATE TABLE "users" (
  "email_address" varchar NOT NULL,
  "prefix" char(8) NOT NULL PRIMARY KEY,
  "token" varchar NOT NULL,
  "generated_at" varchar NOT NULL,
  "expired_at" varchar NOT NULL
);