import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Nav } from "@/components/Nav/Nav";
import { Checkers } from "@/pages/checkers/Checkers";
import { Chess } from "@/pages/chess/Chess";

function App() {
  return (
    <BrowserRouter>
      <Nav />
      <Routes>
        <Route path="/" element={<Navigate to="/checkers" replace />} />
        <Route path="/checkers" element={<Checkers />} />
        <Route path="/chess" element={<Chess />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
