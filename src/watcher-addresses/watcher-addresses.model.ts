import mongoose, { Document, Schema } from "mongoose";

export interface WatcherAddress extends Document {
    watcherId: string;
    address: string;
    lastTx: string;
}

const watcherAddressModel = new Schema<WatcherAddress>(
    {
        watcherId: { type: String, required: true },
        address: { type: String, required: true },
        lastTx: { type: String, required: true }
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

export const WatcherAddressModel = mongoose.model<WatcherAddress>(
    "WatcherAddressModel",
    watcherAddressModel
);