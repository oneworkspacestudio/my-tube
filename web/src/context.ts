import { createContext, useContext } from "react";
export type UIContext = {
  app: any;
  action: (name: string, payload?: any) => void;
  field: (name: string, value: any) => void;
};
export const Context = createContext<UIContext>(null!);
export const useUI = () => useContext(Context);
