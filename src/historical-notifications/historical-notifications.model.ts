import mongoose, { Document, Schema } from "mongoose";

export interface HistoricalNotification extends Document {
  watcherId: string;
  title: string;
  body: string;
  sent: boolean;
  timestamp: number;
}

const historicalNotificationModel = new Schema<HistoricalNotification>(
  {
    watcherId: { type: String, required: true },
    title: { type: String, required: true },
    body: { type: String, required: true },
    sent: { type: Boolean, required: true },
    timestamp: { type: Number, required: true }
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

export const HistoricalNotificationModel = mongoose.model<HistoricalNotification>(
  "HistoricalNotificationModel",
  historicalNotificationModel
);