import { expect, test, type Locator, type Page } from '@playwright/test';

type FieldType = 'string' | 'number' | 'date' | 'boolean';

type FieldDefinition = {
    name: string;
    label: string;
    type: FieldType;
    createValue: string | boolean;
    validationMessage?: string;
    useTimestamp?: boolean;
};

type ModelConfig = {
    name: string;
    plural: string;
    route: string;
    createRoute: string;
    createLinkLabel: string;
    saveButtonLabel: string;
    deleteButtonLabel: string;
    listHeaders: string[];
    toastMessages: {
        updateSuccess: string;
    };
    fields: FieldDefinition[];
    createAssertField: string;
    editScenario: {
        fieldName: string;
        newValue: string | boolean;
    };
};

const modelConfig: ModelConfig = {
    // GF_MODEL_CONFIG_START
    name: 'Skeleton',
    plural: 'Skeletons',
    route: '/models/skeletons',
    createRoute: '/models/skeletons/new',
    createLinkLabel: 'Create New Skeleton',
    saveButtonLabel: 'Save',
    deleteButtonLabel: 'Delete',
    listHeaders: [
        'Name',
        'Age',
        'Death',
        'Zombie',
        'Created',
        'Updated',
    ],
    toastMessages: {
        updateSuccess: 'Skeleton updated successfully.',
    },
    fields: [
        {
            name: 'name',
            label: 'Name',
            type: 'string',
            createValue: 'Test Skeleton',
            validationMessage: 'Enter at least 3 characters',
            useTimestamp: true,
        },
        {
            name: 'age',
            label: 'Age',
            type: 'number',
            createValue: '100',
            validationMessage: 'Enter a positive number',
        },
        {
            name: 'death',
            label: 'Death',
            type: 'date',
            createValue: '2025-01-01',
            validationMessage: 'Select a valid date',
        },
        {
            name: 'zombie',
            label: 'Zombie',
            type: 'boolean',
            createValue: true,
        },
    ],
    createAssertField: 'name',
    editScenario: {
        fieldName: 'name',
        newValue: 'Edited Skeleton',
    },
    // GF_MODEL_CONFIG_END
};

type FieldValues = Record<string, string | boolean>;

function getFieldControl(page: Page, field: FieldDefinition): Locator {
    return page.getByLabel(field.label);
}

async function setControlValue(
    control: Locator,
    field: FieldDefinition,
    value: string | boolean,
): Promise<void> {
    if (field.type === 'boolean') {
        if (value) {
            await control.check();
        } else {
            await control.uncheck();
        }
        return;
    }

    await control.fill(String(value));
}

async function expectControlValue(
    control: Locator,
    field: FieldDefinition,
    value: string | boolean,
): Promise<void> {
    if (field.type === 'boolean') {
        if (value) {
            await expect(control).toBeChecked();
        } else {
            await expect(control).not.toBeChecked();
        }
        return;
    }

    await expect(control).toHaveValue(String(value));
}

function formatListValue(field: FieldDefinition, value: string | boolean): string {
    if (field.type === 'boolean') {
        return value ? 'Yes' : 'No';
    }

    if (field.type === 'date') {
        return new Date(String(value)).toLocaleDateString();
    }

    return String(value);
}

function buildCreateValues(fields: FieldDefinition[]): FieldValues {
    const values: FieldValues = {};
    const timestamp = Date.now();

    for (const field of fields) {
        const rawValue = field.createValue;
        if (field.useTimestamp && typeof rawValue === 'string') {
            values[field.name] = `${rawValue} ${timestamp}`;
        } else {
            values[field.name] = rawValue;
        }
    }

    return values;
}

async function fillForm(page: Page, values: FieldValues): Promise<void> {
    for (const field of modelConfig.fields) {
        const control = getFieldControl(page, field);
        await setControlValue(control, field, values[field.name]);
    }
}

test.describe(modelConfig.plural, () => {
    test.beforeEach(async ({ page }) => {
        await page.goto(modelConfig.route);
    });

    test('should allow a user to list entries', async ({ page }) => {
        await expect(page.getByRole('heading', { name: modelConfig.plural })).toBeVisible();
        for (const header of modelConfig.listHeaders) {
            await expect(page.getByRole('columnheader', { name: header })).toBeVisible();
        }
    });

    test('should allow a user to add an entry', async ({ page }) => {
        await page.getByRole('link', { name: modelConfig.createLinkLabel }).click();
        await expect(page).toHaveURL(new RegExp(`.*${modelConfig.createRoute}`));

        const createValues = buildCreateValues(modelConfig.fields);
        await fillForm(page, createValues);
        await page.getByRole('button', { name: modelConfig.saveButtonLabel }).click();

        await expect(page.getByRole('heading', { name: modelConfig.plural })).toBeVisible();

        const assertField = modelConfig.fields.find(
            (field) => field.name === modelConfig.createAssertField,
        );
        if (assertField) {
            const expected = formatListValue(assertField, createValues[assertField.name]);
            // Use .first() to handle cases where multiple cells have the same value (e.g., multiple bool columns with "Yes")
            await expect(
                page.getByRole('cell', { name: expected, exact: true }).first(),
            ).toBeVisible();
        }
    });

    test('should allow a user to edit an entry', async ({ page }) => {
        await page.getByRole('link', { name: 'Edit' }).first().click();
        await expect(page).toHaveURL(
            new RegExp(`.*${modelConfig.route}/[a-zA-Z0-9-]+`),
        );
        const url = page.url();

        const editField = modelConfig.fields.find(
            (field) => field.name === modelConfig.editScenario.fieldName,
        );
        if (!editField) {
            throw new Error(
                `Configured edit field "${modelConfig.editScenario.fieldName}" was not found`,
            );
        }

        const control = getFieldControl(page, editField);
        await setControlValue(control, editField, modelConfig.editScenario.newValue);
        await page.getByRole('button', { name: modelConfig.saveButtonLabel }).click();

        await expect(page).toHaveURL(url);
        await expect(
            page.getByText(modelConfig.toastMessages.updateSuccess),
        ).toBeVisible();
        await expectControlValue(control, editField, modelConfig.editScenario.newValue);
    });

    test('should show validation hints on empty form', async ({ page }) => {
        await page.getByRole('link', { name: modelConfig.createLinkLabel }).click();
        await expect(page).toHaveURL(new RegExp(`.*${modelConfig.createRoute}`));

        await page.getByRole('button', { name: modelConfig.saveButtonLabel }).click();

        const validationTotals = modelConfig.fields.reduce<Record<string, number>>(
            (counts, field) => {
                if (!field.validationMessage) {
                    return counts;
                }
                counts[field.validationMessage] =
                    (counts[field.validationMessage] ?? 0) + 1;
                return counts;
            },
            {},
        );

        for (const [message, count] of Object.entries(validationTotals)) {
            await expect(page.getByText(message)).toHaveCount(count);
        }
    });

    test('should allow a user to remove an entry', async ({ page }) => {
        page.on('dialog', (dialog) => dialog.accept());
        await page.getByRole('button', { name: modelConfig.deleteButtonLabel }).first().click();
        await expect(page.getByRole('heading', { name: modelConfig.plural })).toBeVisible();
    });
});
