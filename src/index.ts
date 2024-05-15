import express from "express";
import mongoose from "mongoose";
import cors from "cors";
import {dbUrl} from "./config";

async function main() {
    if (!dbUrl) {
        throw new Error("DB URL not found");
    }

    await mongoose.connect(dbUrl);

    const app = express();
    app.use(cors());
    app.use(express.json());
    app.use(express.urlencoded({extended: true}));
}

main().then((res) => console.log(res));