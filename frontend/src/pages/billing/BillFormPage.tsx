import AddRoundedIcon from "@mui/icons-material/AddRounded";
import DeleteRoundedIcon from "@mui/icons-material/DeleteRounded";
import { zodResolver } from "@hookform/resolvers/zod";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { Controller, useFieldArray, useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { useState } from "react";

import AsyncAutocomplete from "@/components/common/AsyncAutocomplete";
import {
  BILL_ITEM_CATEGORIES,
  billFormSchema,
  emptyBillFormValues,
  emptyBillItem,
  type BillFormInput,
} from "@/features/billing/billSchema";
import { billingApi } from "@/features/billing/billingApi";
import { patientsApi } from "@/features/patients/patientsApi";
import { ApiRequestError } from "@/services/apiClient";

const CATEGORY_LABELS: Record<(typeof BILL_ITEM_CATEGORIES)[number], string> = {
  CONSULTATION: "Consultation",
  LAB: "Lab",
  PHARMACY: "Pharmacy",
  PROCEDURE: "Procedure",
  ROOM_CHARGE: "Room Charge",
  OTHER: "Other",
};

export default function BillFormPage() {
  const navigate = useNavigate();
  const [submitError, setSubmitError] = useState<string | null>(null);

  const {
    control,
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<BillFormInput>({
    resolver: zodResolver(billFormSchema),
    defaultValues: emptyBillFormValues,
  });

  const { fields, append, remove } = useFieldArray({ control, name: "items" });

  const patientLabel = watch("patientLabel");
  const items = watch("items");
  const discount = Number(watch("discount")) || 0;
  const tax = Number(watch("tax")) || 0;

  const subtotal = items.reduce((sum, item) => sum + (Number(item.quantity) || 0) * (Number(item.unitPrice) || 0), 0);
  const totalAmount = Math.max(subtotal - discount + tax, 0);

  const onSubmit = async (values: BillFormInput) => {
    setSubmitError(null);
    try {
      const bill = await billingApi.create({
        patientId: values.patientId,
        items: values.items.map((i) => ({
          description: i.description,
          category: i.category,
          quantity: i.quantity,
          unitPrice: i.unitPrice,
        })),
        discount: values.discount,
        tax: values.tax,
      });
      navigate(`/billing/${bill.id}`);
    } catch (err) {
      setSubmitError(err instanceof ApiRequestError ? err.message : "Failed to create bill.");
    }
  };

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} sx={{ mb: 3 }}>
        New Bill
      </Typography>

      {submitError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {submitError}
        </Alert>
      )}

      <Paper variant="outlined" sx={{ p: 3, maxWidth: 900 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid item xs={12} sm={6}>
              <AsyncAutocomplete
                label="Patient"
                required
                value={patientLabel ? { id: watch("patientId"), label: patientLabel } : null}
                onChange={(option) => {
                  setValue("patientId", option?.id ?? "");
                  setValue("patientLabel", option?.label ?? "");
                }}
                error={!!errors.patientId}
                helperText={errors.patientId?.message}
                fetchOptions={async (query) => {
                  const result = await patientsApi.search({ query, pageSize: 10 });
                  return result.items.map((p) => ({ id: p.id, label: `${p.name} (${p.patientCode})` }));
                }}
              />
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />
          <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
            <Typography variant="subtitle2" color="text.secondary">
              Charges
            </Typography>
            <Button size="small" startIcon={<AddRoundedIcon />} onClick={() => append(emptyBillItem)}>
              Add Charge
            </Button>
          </Stack>
          {errors.items?.message && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {errors.items.message}
            </Alert>
          )}

          <Stack spacing={2} sx={{ mb: 3 }}>
            {fields.map((field, index) => (
              <Paper key={field.id} variant="outlined" sx={{ p: 2 }}>
                <Grid container spacing={2} alignItems="flex-start">
                  <Grid item xs={12} sm={4}>
                    <TextField
                      label="Description"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.description}
                      helperText={errors.items?.[index]?.description?.message}
                      {...register(`items.${index}.description`)}
                    />
                  </Grid>
                  <Grid item xs={12} sm={3}>
                    <Controller
                      name={`items.${index}.category`}
                      control={control}
                      render={({ field: controllerField }) => (
                        <FormControl fullWidth required error={!!errors.items?.[index]?.category}>
                          <InputLabel id={`category-label-${index}`}>Category</InputLabel>
                          <Select labelId={`category-label-${index}`} label="Category" {...controllerField}>
                            {BILL_ITEM_CATEGORIES.map((c) => (
                              <MenuItem key={c} value={c}>
                                {CATEGORY_LABELS[c]}
                              </MenuItem>
                            ))}
                          </Select>
                          {errors.items?.[index]?.category && (
                            <FormHelperText>{errors.items[index]?.category?.message}</FormHelperText>
                          )}
                        </FormControl>
                      )}
                    />
                  </Grid>
                  <Grid item xs={6} sm={2}>
                    <TextField
                      label="Quantity"
                      type="number"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.quantity}
                      helperText={errors.items?.[index]?.quantity?.message}
                      {...register(`items.${index}.quantity`)}
                    />
                  </Grid>
                  <Grid item xs={6} sm={2}>
                    <TextField
                      label="Unit Price"
                      type="number"
                      fullWidth
                      required
                      error={!!errors.items?.[index]?.unitPrice}
                      helperText={errors.items?.[index]?.unitPrice?.message}
                      {...register(`items.${index}.unitPrice`)}
                    />
                  </Grid>
                  <Grid item xs={6} sm={1}>
                    <IconButton
                      color="error"
                      onClick={() => remove(index)}
                      disabled={fields.length === 1}
                      aria-label="Remove charge"
                    >
                      <DeleteRoundedIcon />
                    </IconButton>
                  </Grid>
                </Grid>
              </Paper>
            ))}
          </Stack>

          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={6} sm={3}>
              <TextField
                label="Discount"
                type="number"
                fullWidth
                error={!!errors.discount}
                helperText={errors.discount?.message}
                {...register("discount")}
              />
            </Grid>
            <Grid item xs={6} sm={3}>
              <TextField label="Tax" type="number" fullWidth error={!!errors.tax} helperText={errors.tax?.message} {...register("tax")} />
            </Grid>
          </Grid>

          <Paper variant="outlined" sx={{ p: 2, mb: 3, bgcolor: "action.hover" }}>
            <Stack spacing={0.5}>
              <Stack direction="row" justifyContent="space-between">
                <Typography variant="body2" color="text.secondary">
                  Subtotal
                </Typography>
                <Typography variant="body2">{subtotal.toFixed(2)}</Typography>
              </Stack>
              <Stack direction="row" justifyContent="space-between">
                <Typography variant="body2" color="text.secondary">
                  Discount
                </Typography>
                <Typography variant="body2">-{discount.toFixed(2)}</Typography>
              </Stack>
              <Stack direction="row" justifyContent="space-between">
                <Typography variant="body2" color="text.secondary">
                  Tax
                </Typography>
                <Typography variant="body2">+{tax.toFixed(2)}</Typography>
              </Stack>
              <Divider sx={{ my: 0.5 }} />
              <Stack direction="row" justifyContent="space-between">
                <Typography variant="subtitle1" fontWeight={700}>
                  Total
                </Typography>
                <Typography variant="subtitle1" fontWeight={700}>
                  {totalAmount.toFixed(2)}
                </Typography>
              </Stack>
            </Stack>
          </Paper>

          <Stack direction="row" spacing={2} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : "Create Bill"}
            </Button>
          </Stack>
        </Box>
      </Paper>
    </Box>
  );
}
