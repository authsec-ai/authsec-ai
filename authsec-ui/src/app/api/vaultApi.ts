import { baseApi } from './baseApi';

// Placeholder API for vault/secrets feature - will be implemented when backend is ready
export const vaultApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Placeholder endpoints that return mock data for now
    getSecrets: builder.query<any[], any>({
      queryFn: () => ({ data: [] }), // Return empty array for now
      providesTags: ['Secret'],
    }),

    getSecret: builder.query<any, string>({
      queryFn: () => ({ data: null }),
      providesTags: (result, error, id) => [{ type: 'Secret', id }],
    }),

    createSecret: builder.mutation<any, any>({
      queryFn: () => ({ data: null }),
      invalidatesTags: ['Secret'],
    }),

    updateSecret: builder.mutation<any, { id: string; data: any }>({
      queryFn: () => ({ data: null }),
      invalidatesTags: (result, error, { id }) => [{ type: 'Secret', id }],
    }),

    deleteSecret: builder.mutation<void, string>({
      queryFn: () => ({ data: undefined }),
      invalidatesTags: ['Secret'],
    }),

    importSecrets: builder.mutation<any, any>({
      queryFn: () => ({ data: { imported: 0, failed: 0 } }),
      invalidatesTags: ['Secret'],
    }),
  }),
});

export const {
  useGetSecretsQuery,
  useGetSecretQuery,
  useCreateSecretMutation,
  useUpdateSecretMutation,
  useDeleteSecretMutation,
  useImportSecretsMutation,
} = vaultApi;