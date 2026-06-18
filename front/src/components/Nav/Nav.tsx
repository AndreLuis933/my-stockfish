import { NavLink } from "react-router-dom";
import styles from "./Nav.module.css";

export const Nav = () => (
  <nav className={styles.nav}>
    <NavLink
      to="/checkers"
      className={({ isActive }) => `${styles.link} ${isActive ? styles.active : ""}`}
    >
      Damas
    </NavLink>
    <NavLink
      to="/chess"
      className={({ isActive }) => `${styles.link} ${isActive ? styles.active : ""}`}
    >
      Xadrez
    </NavLink>
  </nav>
);
