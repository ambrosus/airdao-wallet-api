import {ethers} from 'ethers';

// todo ask Taras if front-end depends on this error format or is it just human readable
interface ValidationResult {
    field: string;
    message: string;
}

function validateData(data: any): ValidationResult[] {
    const errors: ValidationResult[] = [];

    if (!data.addresses || !Array.isArray(data.addresses) || data.addresses.length === 0) {
        errors.push({ field: 'addresses', message: 'Addresses are required' });
    } else {
        for (const address of data.addresses) {
            if (typeof address !== 'string' || !isValidAddress(address)) {
                errors.push({ field: 'addresses', message: 'Invalid address' });
                break;
            }
        }
    }

    if (data.threshold === undefined || !isValidThreshold(data.threshold)) {
        errors.push({ field: 'threshold', message: 'Invalid threshold (can be 5, 8 or 10)' });
    }

    if (data.notification !== 'on' && data.notification !== 'off') {
        errors.push({ field: 'notification', message: 'Invalid notification (can be "on" or "off")' });
    }

    return errors;
}

function isValidAddress(address: string): boolean {
    return ethers.isAddress(address);
}

function isValidThreshold(threshold: number): boolean {
    return [5, 8, 10].includes(threshold);
}

// Usage
function testValidation() {
    const data = {
        addresses: ['address1', 'address2'],
        threshold: 5,
        notification: 'on'
    };

    const validationErrors = validateData(data);
    if (validationErrors.length > 0) {
        console.log("Validation failed:");
        validationErrors.forEach(error => console.log(`${error.field}: ${error.message}`));
    } else {
        console.log("Validation successful.");
    }
}

// testValidation();
