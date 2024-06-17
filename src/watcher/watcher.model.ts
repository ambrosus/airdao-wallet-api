import mongoose, { Document, Schema } from "mongoose";

export interface Watcher extends Document {
    deviceId?: string;
    pushToken: string;
    threshold: number;
    tokenPrice?: number;
    txNotification: string;
    priceNotification: string;
    addresses?: string[];
    lastSuccessDate?: number;
    lastFailDate?: number;
}

const watcherModel = new Schema<Watcher>(
    {
        deviceId: { type: String, required: false },
        pushToken: { type: String, required: true },
        threshold: {
            type: Number,
            required: true,
            enum: [5, 8, 10],
            message: "incorrect threshold (can be 5, 8 or 10)",
        },
        tokenPrice: { type: Number, required: false },
        txNotification: {
            type: String,
            required: true,
            enum: ["ON", "OFF"],
            message: "txNotification must be either ON or OFF.",
        },
        priceNotification: {
            type: String,
            required: true,
            enum: ["ON", "OFF"],
            message: "priceNotification must be either ON or OFF.",
        },
        addresses: { type: [String], required: false },
        lastSuccessDate: { type: Number, required: false },
        lastFailDate: { type: Number, required: false },
    },
    {
        timestamps: true,
        toJSON: {
            virtuals: true,
            transform: (obj, ret) => {
                delete ret._id;
                delete ret.__v;
                delete ret.createdAt;
                delete ret.updatedAt;
            },
        },
    }
);

export const WatcherModel = mongoose.model<Watcher>(
    "WatcherModel",
    watcherModel
);