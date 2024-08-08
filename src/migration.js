// eslint-disable-next-line @typescript-eslint/no-var-requires,no-undef
const mongoose = require("mongoose");
// eslint-disable-next-line @typescript-eslint/no-var-requires,no-undef
const dotenv = require("dotenv");

const { Schema } = mongoose;

const CHUNK_SIZE = 10;

dotenv.config();

const oldWatcherSchema = new Schema({
    deviceId: String,
    pushToken: String,
    threshold: Number,
    tokenPrice: Number,
    txNotification: String,
    priceNotification: String,
    addresses: [{
        address: String,
        lastTx: String,
    }],
    historicalNotifications: [{
        title: String,
        body: String,
        sent: Boolean,
        timestamp: Date,
    }],
    lastSuccessDate: Date,
    lastFailDate: Date,
    createdAt: Date,
    updatedAt: Date,
}, { collection: "watcher", modelName: "watcher" });

const OldWatcher = mongoose.model("watcher", oldWatcherSchema);

const watcherSchema = new Schema({
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
        type: String, required: true,
        enum: ["ON", "OFF"],
        message: "txNotification must be either ON or OFF.",
    },
    priceNotification: {
        type: String, required: true,
        enum: ["ON", "OFF"],
        message: "priceNotification must be either ON or OFF.",
    },
    lastSuccessDate: { type: Number, required: false },
    lastFailDate: { type: Number, required: false },
}, {
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
});

const Watcher = mongoose.model("WatcherModel", watcherSchema);

const watcherAddressSchema = new Schema({
    watcherId: { type: String, required: true },
    address: { type: String, required: true },
    lastTx: { type: String, required: false }
}, {
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
});

const WatcherAddress = mongoose.model("WatcherAddressModel", watcherAddressSchema);

const historicalNotificationSchema = new Schema({
    watcherId: { type: String, required: true },
    title: { type: String, required: true },
    body: { type: String, required: true },
    sent: { type: Boolean, required: true },
    timestamp: { type: Number, required: true }
}, {
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
});

const HistoricalNotification = mongoose.model("HistoricalNotificationModel", historicalNotificationSchema);

async function processChunk(chunk) {
    console.log("processing chunk...");
    for (const oldWatcher of chunk) {
        const newWatcher = new Watcher({
            deviceId: oldWatcher.deviceId,
            pushToken: oldWatcher.pushToken,
            threshold: oldWatcher.threshold,
            tokenPrice: oldWatcher.tokenPrice,
            txNotification: oldWatcher.txNotification,
            priceNotification: oldWatcher.priceNotification,
            lastSuccessDate: oldWatcher.lastSuccessDate ? oldWatcher.lastSuccessDate.getTime() : undefined,
            lastFailDate: oldWatcher.lastFailDate ? oldWatcher.lastFailDate.getTime() : undefined,
        });

        await newWatcher.save();

        for (const address of oldWatcher.addresses) {
            const newAddress = new WatcherAddress({
                watcherId: newWatcher._id.toString(),
                address: address.address,
                lastTx: address.lastTx,
            });
            await newAddress.save();
        }

        for (const notification of oldWatcher.historicalNotifications) {
            const newNotification = new HistoricalNotification({
                watcherId: newWatcher._id.toString(),
                title: notification.title,
                body: notification.body,
                sent: notification.sent,
                timestamp: notification.timestamp.getTime(),
            });
            await newNotification.save();
        }

        console.log("chunk processed");
    }
}

async function migrateData() {
    const connectionString = process.env.MONGO_DB_URL;

    const sourceDbUrl = connectionString.replace("AIRDAO-MOBILE", "AIRDAO-MOBILE-OLD");

    await mongoose.connect(sourceDbUrl, { useNewUrlParser: true, useUnifiedTopology: true });

    let skip = 0;
    let hasMore = true;

    while (hasMore) {
        console.log("inside iteration");
        const chunk = await OldWatcher.find({}).skip(skip).limit(CHUNK_SIZE);
        console.log("chunk length", chunk.length);
        if (chunk.length > 0) {
            await processChunk(chunk);
            skip += CHUNK_SIZE;
        } else {
            hasMore = false;
        }
    }

    await mongoose.disconnect();
}

migrateData().then(() => {
    console.log("Migration completed successfully");
}).catch(err => {
    console.error("Migration failed", err);
});
